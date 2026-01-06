package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"
	"github.com/xuri/excelize/v2"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type ParticipanteHandler struct {
	DB *sql.DB
}

func NewParticipanteHandler(db *sql.DB) *ParticipanteHandler {
	return &ParticipanteHandler{DB: db}
}

func (h *ParticipanteHandler) Criar(c echo.Context) error {
	type Request struct {
		EmpresaID int    `json:"empresa_id"`
		Logins    string `json:"logins"`
		Senha     string `json:"senha"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	if req.EmpresaID == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "empresa_id é obrigatório"})
	}

	if req.Senha == "" {
		req.Senha = "rumo" + time.Now().Format("2006")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Senha), bcrypt.DefaultCost)

	logins := strings.FieldsFunc(req.Logins, func(r rune) bool {
		return r == ';' || r == ','
	})

	var jaCadastrados []string

	for _, login := range logins {
		login = strings.TrimSpace(login)

		var exists int
		h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE login = ?`, login).Scan(&exists)

		if exists > 0 {
			jaCadastrados = append(jaCadastrados, login)
			continue
		}

		_, err := h.DB.Exec(`
			INSERT INTO users (name, login, type, id_empresa, password)
			VALUES ('', ?, 2, ?, ?)
		`, login, req.EmpresaID, hash)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{"error": "erro ao criar usuário"})
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"status":          "ok",
		"ja_cadastrados":  jaCadastrados,
	})
}

func (h *ParticipanteHandler) Lista(c echo.Context) error {
	empresaID := c.Param("id")

	rows, err := h.DB.Query(`
		SELECT id, login, termo_consentimento, log
		FROM users
		WHERE id_empresa = ? AND type = 2
	`, empresaID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "erro ao buscar participantes"})
	}
	defer rows.Close()

	type Participante struct {
		ID                int    `json:"id"`
		Login             string `json:"login"`
		TermoConsentimento string `json:"termo_consentimento"`
		Log               string `json:"log"`
	}

	var participantes []Participante

	for rows.Next() {
		var p Participante
		rows.Scan(&p.ID, &p.Login, &p.TermoConsentimento, &p.Log)
		participantes = append(participantes, p)
	}

	return c.JSON(http.StatusOK, participantes)
}

func (h *ParticipanteHandler) GeraIndices(c echo.Context) error {
	rows, err := h.DB.Query(`
		SELECT
			pr.id_user,
			pr.id_programa,
			pr.flag,
			pr.s_n,
			a.alternativa
		FROM participantes_respostas pr
			JOIN perguntas p ON (p.id = pr.id_pergunta)
			LEFT JOIN perguntas pdep ON (pdep.id_dependente = pr.id)
			LEFT JOIN participantes_respostas prdep ON (prdep.id_pergunta = pdep.id)
			LEFT JOIN alternativas a ON (a.id = prdep.id_alternativa_resposta)
		WHERE pr.id_questionario = 95
	`)
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao buscar respostas"})
	}
	defer rows.Close()

	type Key struct {
		UserID     int
		ProgramaID int
	}

	type Contador struct {
		Indice int
		Comp   int
	}

	contadores := make(map[Key]*Contador)

	for rows.Next() {
		var userID, programaID int
		var flag, sn sql.NullString
		var alternativa sql.NullString

		rows.Scan(&userID, &programaID, &flag, &sn, &alternativa)

		if !flag.Valid || !sn.Valid || sn.String != "s" {
			continue
		}

		if alternativa.String != "algumas vezes" && alternativa.String != "muitas vezes" {
			continue
		}

		key := Key{userID, programaID}
		if _, ok := contadores[key]; !ok {
			contadores[key] = &Contador{}
		}

		if flag.String == "depressao_indice" {
			contadores[key].Indice++
		}
		if flag.String == "depressao_comp" {
			contadores[key].Comp++
		}
	}

	for key, cts := range contadores {
		depressao := "nao"
		if cts.Indice >= 1 && cts.Comp >= 3 {
			depressao = "sim"
		}

		termometro := h.calculaTermometro(key.UserID)
		cor, bg := h.calculaTermometroCor(termometro)

		_, err := h.DB.Exec(`
			UPDATE participantes_questionarios
				SET depressao = ?, termometro_cor = ?, termometro_bg = ?
			WHERE id_user = ? AND id_programa = ? AND id_questionario = 95
		`, depressao, cor, bg, key.UserID, key.ProgramaID)

		if err != nil {
			return c.JSON(500, echo.Map{"error": "erro ao atualizar índices"})
		}
	}

	return c.JSON(200, echo.Map{"status": "indices recalculados"})
}

func (h *ParticipanteHandler) calculaTermometro(userID int) int {
	rows, _ := h.DB.Query(`
		SELECT p.tipo_pergunta, pr.s_n, pr.s_pontuacao, pr.n_pontuacao
		FROM participantes_respostas pr
			JOIN perguntas p ON (p.id = pr.id_pergunta)
		WHERE pr.id_user = ? AND pr.id_questionario = 95
	`, userID)
	defer rows.Close()

	total := 0

	for rows.Next() {
		var tipo, sn string
		var sp, np int
		rows.Scan(&tipo, &sn, &sp, &np)

		if tipo == "sn" {
			if sn == "s" {
				total += sp
			} else {
				total += np
			}
		}
	}

	return total
}

func (h *ParticipanteHandler) calculaTermometroCor(pontos int) (string, string) {
	row := h.DB.QueryRow(`
		SELECT legenda, cor
		FROM questionarios_intervalos
		WHERE id_questionario = 95 AND ? BETWEEN intervalo_inicio AND intervalo_termino
		LIMIT 1
	`, pontos)

	var cod, bg string
	row.Scan(&cod, &bg)

	return cod, bg
}

func (h *ParticipanteHandler) Termometro(c echo.Context) error {
	userID := c.Param("id")

	rows, _ := h.DB.Query(`
		SELECT tipo_pergunta, s_n, s_pontuacao, n_pontuacao
		FROM participantes_respostas pr
		JOIN perguntas p ON p.id = pr.id_pergunta
		WHERE pr.id_user = ?
			AND pr.id_questionario = 95
	`, userID)
	defer rows.Close()

	total := 0

	for rows.Next() {
		var tipo, sn string
		var sp, np int
		rows.Scan(&tipo, &sn, &sp, &np)

		if tipo == "sn" {
			if sn == "s" {
				total += sp
			} else {
				total += np
			}
		}
	}

	return c.JSON(http.StatusOK, echo.Map{"pontos": total})
}

func (h *ParticipanteHandler) TermometroCor(c echo.Context) error {
	pontos := c.QueryParam("pontos")

	row := h.DB.QueryRow(`
		SELECT legenda, cor
		FROM questionarios_intervalos
		WHERE id_questionario = 95
			AND ? BETWEEN intervalo_inicio AND intervalo_termino
		LIMIT 1
	`, pontos)

	var legenda, cor string
	row.Scan(&legenda, &cor)

	return c.JSON(http.StatusOK, echo.Map{
		"cod": legenda,
		"bg":  cor,
	})
}

func (h *ParticipanteHandler) Apagar(c echo.Context) error {
	id := c.Param("id")

	_, err := h.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "erro ao excluir"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ParticipanteHandler) ApagarPorEmpresa(c echo.Context) error {
	empresaID := c.Param("id")

	_, err := h.DB.Exec(`DELETE FROM users WHERE id_empresa = ?`, empresaID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "erro ao excluir participantes"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *ParticipanteHandler) UploadExcel(c echo.Context) error {
	empresaID := c.FormValue("empresa_id")
	senha := c.FormValue("senha")

	if empresaID == "" {
		return c.JSON(400, echo.Map{"error": "empresa_id obrigatório"})
	}

	if senha == "" {
		senha = "rumo" + time.Now().Format("2006")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)

	file, err := c.FormFile("excel")
	if err != nil {
		return c.JSON(400, echo.Map{"error": "arquivo obrigatório"})
	}

	src, _ := file.Open()
	defer src.Close()

	xl, err := excelize.OpenReader(src)
	if err != nil {
		return c.JSON(400, echo.Map{"error": "arquivo inválido"})
	}

	rows, _ := xl.GetRows(xl.GetSheetName(0))

	var jaCadastrados []string

	for i, row := range rows {
		if i == 0 || len(row) == 0 {
			continue
		}

		login := strings.TrimSpace(row[0])

		var exists int
		h.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE login = ?`, login).Scan(&exists)

		if exists > 0 {
			jaCadastrados = append(jaCadastrados, login)
			continue
		}

		h.DB.Exec(`
			INSERT INTO users (name, login, type, id_empresa, password)
			VALUES ('', ?, 2, ?, ?)
		`, login, empresaID, hash)
	}

	return c.JSON(200, echo.Map{
		"status":         "ok",
		"ja_cadastrados": jaCadastrados,
	})
}


