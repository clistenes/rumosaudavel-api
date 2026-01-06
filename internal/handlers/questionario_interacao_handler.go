package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type QuestionarioInteracaoHandler struct {
	DB *sql.DB
}

func NewQuestionarioInteracaoHandler(db *sql.DB) *QuestionarioInteracaoHandler {
	return &QuestionarioInteracaoHandler{DB: db}
}

func (h *QuestionarioInteracaoHandler) Home(c echo.Context) error {
	userID := c.Get("userID").(int)

	var cadastrado sql.NullString
	var empresaID int

	err := h.DB.QueryRow(`
		SELECT cadastrado, id_empresa FROM users WHERE id = ?
	`, userID).Scan(&cadastrado, &empresaID)

	if err != nil || !cadastrado.Valid {
		return c.Redirect(http.StatusFound, "/boas-vindas")
	}

	rows, _ := h.DB.Query(`
		SELECT p.id
		FROM programas_empresas pe
			JOIN programas p ON p.id = pe.id_programa
		WHERE pe.id_empresa = ?
	`, empresaID)
	defer rows.Close()

	total := 0
	for rows.Next() {
		total++
	}

	if total == 0 {
		return c.Redirect(http.StatusFound, "/sem-vinculo-programa")
	}

	return c.JSON(http.StatusOK, echo.Map{
		"programas": total,
	})
}

func (h *QuestionarioInteracaoHandler) Questionario(c echo.Context) error {
	userID := c.Get("userID").(int)

	questionarioID, _ := strconv.Atoi(c.QueryParam("questionario"))
	programaID, _ := strconv.Atoi(c.QueryParam("programa"))

	var ultimaPergunta sql.NullInt64
	h.DB.QueryRow(`
		SELECT id_pergunta
			FROM participantes_respostas
		WHERE id_user = ? AND id_questionario = ?
		ORDER BY id DESC LIMIT 1
	`, userID, questionarioID).Scan(&ultimaPergunta)

	var (
		id   int
		nome string
	)

	if !ultimaPergunta.Valid {
		err := h.DB.QueryRow(`
			SELECT id, nome 
			FROM perguntas
			WHERE id_questionario = ?
			AND id_dependente IS NULL
			ORDER BY grupo_pos IS NULL, grupo_pos, pos
			LIMIT 1
		`, questionarioID).Scan(&id, &nome)

		if err != nil {
			h.finalizaQuestionario(userID, questionarioID, programaID)
			return c.JSON(http.StatusOK, echo.Map{"status": "finalizado"})
		}
	} else {
		err := h.DB.QueryRow(`
			SELECT id, nome FROM perguntas
			WHERE id_questionario = ?
			AND pos > (SELECT pos FROM perguntas WHERE id = ?)
			AND id_dependente IS NULL
			ORDER BY grupo_pos IS NULL, grupo_pos, pos
			LIMIT 1
		`, questionarioID, ultimaPergunta.Int64).Scan(&id, &nome)

		if err != nil {
			h.finalizaQuestionario(userID, questionarioID, programaID)
			return c.JSON(http.StatusOK, echo.Map{"status": "finalizado"})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"id":   id,
		"nome": nome,
	})
}

func (h *QuestionarioInteracaoHandler) ProcessaResposta(c echo.Context) error {
	userID := c.Get("userID").(int)

	perguntaID, _ := strconv.Atoi(c.FormValue("pergunta"))
	questionarioID, _ := strconv.Atoi(c.FormValue("questionario"))
	programaID, _ := strconv.Atoi(c.FormValue("programa"))
	alternativa := c.FormValue("alternativa")

	_, err := h.DB.Exec(`
		INSERT INTO participantes_respostas
		(id_user, id_questionario, id_programa, id_pergunta, id_alternativa_resposta, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, questionarioID, programaID, perguntaID, alternativa, time.Now())

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "erro ao salvar resposta",
		})
	}

	return c.Redirect(
		http.StatusFound,
		"/participante/questionario?questionario="+strconv.Itoa(questionarioID)+"&programa="+strconv.Itoa(programaID),
	)
}

func (h *QuestionarioInteracaoHandler) finalizaQuestionario(userID, questionarioID, programaID int) {
	h.DB.Exec(`
		INSERT IGNORE INTO participantes_questionarios
		(id_user, id_questionario, id_programa)
		VALUES (?, ?, ?)
	`, userID, questionarioID, programaID)

	h.geraTermometro(userID, questionarioID)
}

func (h *QuestionarioInteracaoHandler) geraTermometro(userID, questionarioID int) {
	rows, _ := h.DB.Query(`
		SELECT pr.s_n, p.s_pontuacao, p.n_pontuacao
		FROM participantes_respostas pr
		JOIN perguntas p ON p.id = pr.id_pergunta
		WHERE pr.id_user = ? AND pr.id_questionario = ?
	`, userID, questionarioID)
	defer rows.Close()

	total := 0
	for rows.Next() {
		var sn string
		var sP, nP int
		rows.Scan(&sn, &sP, &nP)

		if sn == "s" {
			total += sP
		} else {
			total += nP
		}
	}

	var legenda, cor string
	h.DB.QueryRow(`
		SELECT legenda, cor
		FROM questionarios_intervalos
		WHERE ? BETWEEN intervalo_inicio AND intervalo_termino
		AND id_questionario = ?
	`, total, questionarioID).Scan(&legenda, &cor)

	h.DB.Exec(`
		UPDATE participantes_questionarios
		SET termometro_cor = ?, termometro_bg = ?
		WHERE id_user = ? AND id_questionario = ?
	`, legenda, cor, userID, questionarioID)
}

func (h *QuestionarioInteracaoHandler) Relatorio(c echo.Context) error {
	userID := c.Get("userID").(int)
	id, _ := strconv.Atoi(c.QueryParam("id"))

	rows, _ := h.DB.Query(`
		SELECT p.nome, pr.s_n
		FROM participantes_respostas pr
		JOIN perguntas p ON p.id = pr.id_pergunta
		WHERE pr.id_user = ? AND pr.id_questionario = ?
	`, userID, id)
	defer rows.Close()

	var respostas []echo.Map
	for rows.Next() {
		var nome, sn string
		rows.Scan(&nome, &sn)

		respostas = append(respostas, echo.Map{
			"pergunta": nome,
			"resposta": sn,
		})
	}

	return c.JSON(http.StatusOK, respostas)
}

func (h *QuestionarioInteracaoHandler) Prontuario(c echo.Context) error {
	userID := c.Get("userID").(int)

	rows, _ := h.DB.Query(`
		SELECT data, descricao
		FROM participantes_historico_coaching
		WHERE id_user = ?
	`, userID)
	defer rows.Close()

	var historico []echo.Map
	for rows.Next() {
		var data, desc string
		rows.Scan(&data, &desc)

		historico = append(historico, echo.Map{
			"data": data,
			"descricao": desc,
		})
	}

	return c.JSON(http.StatusOK, historico)
}

func (h *QuestionarioInteracaoHandler) Contato(c echo.Context) error {
	userID := c.Get("userID").(int)

	rows, _ := h.DB.Query(`
		SELECT nome, telefone
		FROM participantes_contatos
		WHERE id_user = ?
	`, userID)
	defer rows.Close()

	var contatos []echo.Map
	for rows.Next() {
		var nome, tel string
		rows.Scan(&nome, &tel)

		contatos = append(contatos, echo.Map{
			"nome":     nome,
			"telefone": tel,
		})
	}

	return c.JSON(http.StatusOK, contatos)
}
