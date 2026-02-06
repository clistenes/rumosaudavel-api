package handlers

import (
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type EmpresaHandler struct {
	DB *sql.DB
}

func NewEmpresaHandler(db *sql.DB) *EmpresaHandler {
	return &EmpresaHandler{DB: db}
}

func (h *EmpresaHandler) Criar(c echo.Context) error {
	termo := "n"
	if c.FormValue("empresa_termo") == "on" {
		termo = "s"
	}

	var logo string
	file, err := c.FormFile("empresa_logotipo")
	if err == nil {
		logo, _ = h.salvarArquivo(file, "uploads/empresas")
	}

	res, err := h.DB.Exec(`
		INSERT INTO empresas (nome, introducao, cor, termo_consentimento, logotipo)
		VALUES (?, ?, ?, ?, ?)
	`, c.FormValue("empresa_nome"), c.FormValue("empresa_introducao"), c.FormValue("empresa_cor"), termo, logo)
	if err != nil {
		return err
	}

	empresaID, _ := res.LastInsertId()

	camposPadrao := c.Request().Form["campos_padrao[]"]
	for _, campo := range camposPadrao {
		h.DB.Exec(`INSERT INTO campos_personalizados_empresas (id_empresa, campo, tipo) VALUES (?, ?, 'text')`, empresaID, campo)
	}

	h.processarCamposExtras(c, int(empresaID))

	return c.Redirect(http.StatusFound, "/adm/lista-empresas")
}

func (h *EmpresaHandler) Lista(c echo.Context) error {
	rows, _ := h.DB.Query("SELECT id, nome, cor FROM empresas")
	defer rows.Close()

	var empresas []echo.Map
	for rows.Next() {
		var id int
		var nome, cor string
		rows.Scan(&id, &nome, &cor)
		empresas = append(empresas, echo.Map{"id": id, "nome": nome, "cor": cor})
	}
	return c.JSON(http.StatusOK, empresas)
}

func (h *EmpresaHandler) Editar(c echo.Context) error {
	id := c.FormValue("empresa_id")
	termo := "n"
	if c.FormValue("empresa_termo") == "on" {
		termo = "s"
	}

	file, err := c.FormFile("empresa_logotipo")
	if err == nil {
		var oldLogo string
		h.DB.QueryRow("SELECT logotipo FROM empresas WHERE id = ?", id).Scan(&oldLogo)
		if oldLogo != "" {
			os.Remove(filepath.Join("public/uploads/empresas", oldLogo))
		}
		newLogo, _ := h.salvarArquivo(file, "uploads/empresas")
		h.DB.Exec("UPDATE empresas SET logotipo = ? WHERE id = ?", newLogo, id)
	}

	_, err = h.DB.Exec(`
		UPDATE empresas SET 
			nome=?, introducao=?, cor=?, termo_consentimento=?, 
			id_campo_dashboard_heatmap_1=?, id_campo_dashboard_heatmap_2=?
		WHERE id=?`,
		c.FormValue("empresa_nome"),
		c.FormValue("empresa_introducao"),
		c.FormValue("empresa_cor"),
		termo,
		c.FormValue("id_campo_dashboard_heatmap_1"),
		c.FormValue("id_campo_dashboard_heatmap_2"),
		id,
	)

	empresaIDInt, _ := strconv.Atoi(id)
	h.processarCamposExtras(c, empresaIDInt)

	return c.Redirect(http.StatusFound, "/adm/editar-empresa/"+id)
}

func (h *EmpresaHandler) Apagar(c echo.Context) error {
	id := c.Param("id")
	tables := []string{"users", "campos_personalizados_empresas", "campos_personalizados_alternativas_empresas", "programas_empresas"}

	for _, table := range tables {
		h.DB.Exec(fmt.Sprintf("DELETE FROM %s WHERE id_empresa = ?", table), id)
	}
	h.DB.Exec("DELETE FROM empresas WHERE id = ?", id)
	return c.Redirect(http.StatusFound, "/adm/lista-empresas")
}

func (h *EmpresaHandler) Dashboard(c echo.Context) error {
	empresaID := c.FormValue("id_empresa")

	var usuariosCount int
	h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id_empresa = ? AND type = '2'", empresaID).Scan(&usuariosCount)

	rows, _ := h.DB.Query(`
		SELECT p.intervalo_inicio, p.intervalo_termino, p.intervalo_tipo 
		FROM programas_empresas pe
		JOIN programas p ON pe.id_programa = p.id
		WHERE pe.id_empresa = ?`, empresaID)
	defer rows.Close()

	campanhasAtivas := 0
	hoje := time.Now().Format("2006-01-02")
	for rows.Next() {
		var inicio, termino sql.NullString
		var tipo string
		rows.Scan(&inicio, &termino, &tipo)
		if (inicio.Valid && hoje >= inicio.String && hoje <= termino.String) || tipo == "indeterminado" {
			campanhasAtivas++
		}
	}

	return c.JSON(http.StatusOK, echo.Map{
		"usuarios":         usuariosCount,
		"campanhas_ativas": campanhasAtivas,
	})
}

func (h *EmpresaHandler) CoachingParticipante(c echo.Context) error {
	idUser := c.FormValue("id_user")
	texto := c.FormValue("texto")
	dataRaw := c.FormValue("data")

	t, _ := time.Parse("02/01/2006", strings.ReplaceAll(dataRaw, "/", "-"))
	dataDB := t.Format("2006-01-02")

	var idHistorico int64
	if idH := c.FormValue("id_historico"); idH != "" {
		h.DB.Exec("UPDATE participantes_historico_coaching SET data=?, texto=? WHERE id=?", dataDB, texto, idH)
		idHistorico, _ = strconv.ParseInt(idH, 10, 64)
	} else {
		res, _ := h.DB.Exec("INSERT INTO participantes_historico_coaching (id_user, data, texto) VALUES (?, ?, ?)", idUser, dataDB, texto)
		idHistorico, _ = res.LastInsertId()
	}

	form, err := c.MultipartForm()
	if err == nil {
		for _, file := range form.File["arquivos"] {
			nomeFinal, _ := h.salvarArquivo(file, "uploads/coaching")
			h.DB.Exec(`INSERT INTO upload_historico_coaching (nome, id_user, id_historico, arquivo) VALUES (?, ?, ?, ?)`,
				c.FormValue("nome_arquivo"), idUser, idHistorico, nomeFinal)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *EmpresaHandler) salvarArquivo(file *multipart.FileHeader, pasta string) (string, error) {
	src, _ := file.Open()
	defer src.Close()

	os.MkdirAll("public/"+pasta, 0755)
	nome := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(file.Filename))
	dstPath := filepath.Join("public", pasta, nome)

	dst, _ := os.Create(dstPath)
	defer dst.Close()

	io.Copy(dst, src)
	return nome, nil
}

func (h *EmpresaHandler) processarCamposExtras(c echo.Context, empresaID int) {
	tipos := c.Request().Form["campoextra[][tipo]"]
	nomes := c.Request().Form["campoextra[][nome]"]
	alts := c.Request().Form["campoextra[][alternativas]"]

	for i := range nomes {
		res, _ := h.DB.Exec("INSERT INTO campos_personalizados_empresas (id_empresa, campo, tipo) VALUES (?, ?, ?)", empresaID, nomes[i], tipos[i])
		if tipos[i] == "radio" && i < len(alts) {
			campoID, _ := res.LastInsertId()
			for _, alt := range strings.Split(alts[i], ";") {
				h.DB.Exec("INSERT INTO campos_personalizados_alternativas_empresas (id_empresa, id_campo, alternativa) VALUES (?, ?, ?)", empresaID, campoID, alt)
			}
		}
	}
}

func (h *EmpresaHandler) ApagarCampo(c echo.Context) error {
	id := c.Param("id")
	var tipo string
	h.DB.QueryRow("SELECT tipo FROM campos_personalizados_empresas WHERE id=?", id).Scan(&tipo)

	if tipo == "radio" {
		h.DB.Exec("DELETE FROM campos_personalizados_alternativas_empresas WHERE id_campo=?", id)
	}
	h.DB.Exec("DELETE FROM campos_personalizados_empresas WHERE id=?", id)
	return c.NoContent(http.StatusOK)
}

func (h *EmpresaHandler) ListaUsuarios(c echo.Context) error {
	empresaid := c.Get("id_empresa")

	rows, _ := h.DB.Query("SELECT id, name, email FROM users WHERE id_empresa = ? AND type = '2'", empresaid)
	defer rows.Close()

	var participantes []echo.Map
	for rows.Next() {
		var id int
		var name, email string
		rows.Scan(&id, &name, &email)
		participantes = append(participantes, echo.Map{"id": id, "name": name, "email": email})
	}
	return c.JSON(http.StatusOK, participantes)
}

func (h *EmpresaHandler) ParticipanteInfo(c echo.Context) error {
	id := c.Param("id")

	respostasRows, _ := h.DB.Query("SELECT campo, resposta FROM campos_empresas_respostas WHERE id_user = ?", id)
	defer respostasRows.Close()
	var respostas []echo.Map
	for respostasRows.Next() {
		var campo, resposta string
		respostasRows.Scan(&campo, &resposta)
		respostas = append(respostas, echo.Map{"campo": campo, "resposta": resposta})
	}

	contatosRows, _ := h.DB.Query("SELECT id, data, contato, tipo_contato FROM participantes_contatos WHERE id_user = ?", id)
	defer contatosRows.Close()
	var contatos []echo.Map
	for contatosRows.Next() {
		var cid int
		var data, contato, tipo string
		contatosRows.Scan(&cid, &data, &contato, &tipo)
		contatos = append(contatos, echo.Map{"id": cid, "data": data, "contato": contato, "tipo": tipo})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"respostas": respostas,
		"contatos":  contatos,
	})
}

func (h *EmpresaHandler) ContatoParticipanteSave(c echo.Context) error {
	dataRaw := c.FormValue("data")
	t, _ := time.Parse("02/01/2006", strings.ReplaceAll(dataRaw, "/", "-"))

	_, err := h.DB.Exec(`
		INSERT INTO participantes_contatos (id_user, data, hora_inicio, hora_final, tipo_contato, contato)
		VALUES (?, ?, ?, ?, ?, ?)`,
		c.FormValue("id_user"),
		t.Format("2006-01-02"),
		c.FormValue("hora_inicio"),
		c.FormValue("hora_final"),
		c.FormValue("tipo_contato"),
		c.FormValue("contato"),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.NoContent(http.StatusCreated)
}

func (h *EmpresaHandler) RelatorioDadosGerais(c echo.Context) error {
	idQuestionario := c.Param("id_questionario")
	idEmpresa := c.Param("id_empresa")
	idPrograma := c.Param("id_programa")

	var totalParticipantes int
	h.DB.QueryRow("SELECT COUNT(*) FROM users WHERE id_empresa = ? AND type = '2'", idEmpresa).Scan(&totalParticipantes)

	var respondentes int
	queryRespondentes := `
		SELECT COUNT(DISTINCT id_user) 
		FROM participantes_questionarios 
		WHERE id_questionario = ? AND id_programa = ? 
		AND id_user IN (SELECT id FROM users WHERE id_empresa = ? AND type = 2)`
	h.DB.QueryRow(queryRespondentes, idQuestionario, idPrograma, idEmpresa).Scan(&respondentes)

	querySexo := `
		SELECT cer.resposta, COUNT(DISTINCT pq.id_user) as total_respondido
		FROM campos_empresas_respostas cer
		LEFT JOIN participantes_questionarios pq ON cer.id_user = pq.id_user 
			AND pq.id_questionario = ? AND pq.id_programa = ?
		WHERE cer.id_empresa = ? AND cer.id_campo = (
			SELECT id FROM campos_empresas WHERE id_empresa = ? AND campo = 'sexo'
		)
		GROUP BY cer.resposta`

	rows, _ := h.DB.Query(querySexo, idQuestionario, idPrograma, idEmpresa, idEmpresa)
	defer rows.Close()

	sexoStats := make(map[string]int)
	for rows.Next() {
		var resp string
		var count int
		rows.Scan(&resp, &count)
		sexoStats[resp] = count
	}

	return c.JSON(http.StatusOK, echo.Map{
		"total_participantes": totalParticipantes,
		"respondentes":        respondentes,
		"por_sexo":            sexoStats,
	})
}

func (h *EmpresaHandler) OrdenaCampos(c echo.Context) error {
	posicoes := c.Request().Form["pos[]"]

	for i, id := range posicoes {
		_, err := h.DB.Exec("UPDATE campos_personalizados_empresas SET pos = ? WHERE id = ?", i+1, id)
		if err != nil {
			return err
		}
	}
	return c.NoContent(http.StatusOK)
}

func (h *EmpresaHandler) GetSemaforoFilterValues(c echo.Context) error {
	filterID := c.QueryParam("filter_id")
	idEmpresa := c.QueryParam("idempresa")

	query := `
		SELECT cer.resposta as name, COUNT(*) as total
		FROM campos_empresas_respostas cer
		WHERE cer.id_empresa = ? AND cer.id_campo = ?
		GROUP BY cer.resposta
		-- HAVING total > 5
		ORDER BY cer.resposta ASC`

	rows, _ := h.DB.Query(query, idEmpresa, filterID)
	defer rows.Close()

	var values []echo.Map
	for rows.Next() {
		var name string
		var total int
		rows.Scan(&name, &total)
		values = append(values, echo.Map{"name": name, "total": total})
	}
	return c.JSON(http.StatusOK, values)
}

func (h *EmpresaHandler) CalculoAnsiedade(c echo.Context) error {
	idUser := c.Param("id_user")
	idQuestionario := c.Param("id_questionario")
	idPrograma := c.Param("id_programa")

	rows, err := h.DB.Query(`
		SELECT pr.flag_ansiedade, a.alternativa 
		FROM participantes_respostas pr
		JOIN alternativas a ON pr.id_alternativa_resposta = a.id
		WHERE pr.id_user = ? AND pr.id_questionario = ? AND pr.id_programa = ?
	`, idUser, idQuestionario, idPrograma)
	if err != nil {
		return err
	}
	defer rows.Close()

	contadorIndice := 0
	contadorComp := 0

	for rows.Next() {
		var flag sql.NullString
		var alternativa string
		rows.Scan(&flag, &alternativa)

		if flag.Valid {
			alt := strings.ToLower(alternativa)
			if flag.String == "ansiedade_indice" {
				if alt == "algumas vezes" || alt == "muitas vezes" {
					contadorIndice++
				}
			}
			if flag.String == "ansiedade_comp" {
				if alt == "algumas vezes" || alt == "muitas vezes" {
					contadorComp++
				}
			}
		}
	}

	if contadorIndice >= 1 && contadorComp >= 3 {
		_, err = h.DB.Exec(`
			UPDATE participantes_questionarios 
			SET ansiedade = 'sim' 
			WHERE id_user = ? AND id_questionario = ? AND id_programa = ?
		`, idUser, idQuestionario, idPrograma)
	}

	return err
}

func (h *EmpresaHandler) RelatorioSomaAlternativasByGroup(c echo.Context) error {
	idQ := c.Param("id_questionario")
	idProg := c.Param("id_programa")
	idEmp := c.Param("id_empresa")
	campoResposta := c.QueryParam("camporesposta")

	query := `
		SELECT 
			p.grupo_nome, 
			p.nome as pergunta_nome, 
			COUNT(pr.id) as total_respostas, 
			AVG(a.pontuacao) as media_pontuacao
		FROM participantes_respostas pr
		JOIN alternativas a ON pr.id_alternativa_resposta = a.id
		JOIN perguntas p ON pr.id_pergunta = p.id
		WHERE pr.id_empresa = ? AND pr.id_programa = ? AND pr.id_questionario = ?
		AND p.tipo = 'me' AND p.id_dependente IS NULL
	`

	var args []interface{}
	args = append(args, idEmp, idProg, idQ)

	if campoResposta != "" {
		query += ` AND pr.id_user IN (
			SELECT id_user FROM campos_empresas_respostas 
			WHERE id_empresa = ? AND resposta = ?
		)`
		args = append(args, idEmp, campoResposta)
	}

	query += " GROUP BY p.grupo_nome, p.nome ORDER BY p.grupo_pos, p.pos"

	rows, _ := h.DB.Query(query, args...)
	defer rows.Close()

	results := make(map[string][]echo.Map)
	for rows.Next() {
		var grupo, pergunta string
		var total int
		var media float64
		rows.Scan(&grupo, &pergunta, &total, &media)
		results[grupo] = append(results[grupo], echo.Map{
			"pergunta": pergunta,
			"total":    total,
			"media":    media,
		})
	}

	return c.JSON(http.StatusOK, results)
}

func (h *EmpresaHandler) GetRiskAnalysis(c echo.Context) error {
	idEmp := c.QueryParam("id_empresa")
	idProg := c.QueryParam("id_programa")
	idQ := c.QueryParam("id_questionario")

	query := `
		SELECT 
			CASE 
				WHEN score <= 25 THEN 'verde'
				WHEN score <= 50 THEN 'amarelo'
				WHEN score <= 75 THEN 'laranja'
				ELSE 'vermelho'
			END as cor,
			COUNT(*) as total
		FROM (
			SELECT pr.id_user, SUM(a.pontuacao) as score
			FROM participantes_respostas pr
			JOIN alternativas a ON pr.id_alternativa_resposta = a.id
			WHERE pr.id_empresa = ? AND pr.id_programa = ? AND pr.id_questionario = ?
			GROUP BY pr.id_user
		) as scores
		GROUP BY cor
	`

	rows, _ := h.DB.Query(query, idEmp, idProg, idQ)
	defer rows.Close()

	analise := make(map[string]int)
	for rows.Next() {
		var cor string
		var total int
		rows.Scan(&cor, &total)
		analise[cor] = total
	}

	return c.JSON(http.StatusOK, analise)
}

func (h *EmpresaHandler) SaveFiltroDashboard(c echo.Context) error {
	idEmpresa := c.FormValue("id_empresa")
	idCampo := c.FormValue("id_campo")

	_, err := h.DB.Exec(`
		INSERT INTO filtro_empresa_dashboard (id_empresa, id_campo) 
		VALUES (?, ?) 
		ON DUPLICATE KEY UPDATE id_campo = ?`,
		idEmpresa, idCampo, idCampo)

	if err != nil {
		return c.String(http.StatusInternalServerError, "Erro ao salvar filtro")
	}
	return c.NoContent(http.StatusOK)
}

func (h *EmpresaHandler) ExportarRelatorioIndividual(c echo.Context) error {
	idUser := c.Param("id")
	idQuest := c.Param("questionario")

	var userName string
	h.DB.QueryRow("SELECT name FROM users WHERE id = ?", idUser).Scan(&userName)

	rows, _ := h.DB.Query(`
		SELECT p.nome, pr.dissertativa, pr.s_n, a.alternativa, p.tipo
		FROM participantes_respostas pr
		JOIN perguntas p ON pr.id_pergunta = p.id
		LEFT JOIN alternativas a ON pr.id_alternativa_resposta = a.id
		WHERE pr.id_user = ? AND pr.id_questionario = ?
		ORDER BY p.pos ASC
	`, idUser, idQuest)
	defer rows.Close()

	// 3. Gerar Documento (Estrutura conceitual)
	// doc := docx.NewFile()
	// addText(doc, "Participante: " + userName)
	// loop rows { ... addText(doc, pergunta + ": " + resposta) ... }

	return c.Attachment("relatorio.docx", "relatorio_"+userName+".docx")
}

type Intervalo struct {
	Legenda string `json:"legenda"`
	Cor     string `json:"cor"`
	Min     int    `json:"min"`
	Max     int    `json:"max"`
}

func (h *EmpresaHandler) GetIntervalos(idQuestionario int) []Intervalo {
	return []Intervalo{
		{"verde", "#0A5", 0, 25},
		{"amarelo", "#FF5", 26, 50},
		{"laranja", "#F80", 51, 75},
		{"vermelho", "#A00", 76, 100},
	}
}

func (h *EmpresaHandler) TermometroCore(idUser, idQuest, idProg int) int {
	var totalPontos int
	query := `
		SELECT SUM(a.pontuacao) 
		FROM participantes_respostas pr
		JOIN alternativas a ON pr.id_alternativa_resposta = a.id
		WHERE pr.id_user = ? AND pr.id_questionario = ? AND pr.id_programa = ?`

	h.DB.QueryRow(query, idUser, idQuest, idProg).Scan(&totalPontos)
	return totalPontos
}

func (h *EmpresaHandler) RelatorioGrafico(c echo.Context) error {
	idQ, _ := strconv.Atoi(c.Param("id_questionario"))
	idEmp, _ := strconv.Atoi(c.Param("id_empresa"))
	idProg, _ := strconv.Atoi(c.Param("id_programa"))

	intervalos := h.GetIntervalos(idQ)

	rows, _ := h.DB.Query(`
		SELECT DISTINCT id_user FROM participantes_questionarios 
		WHERE id_questionario = ? AND id_programa = ? 
		AND id_user IN (SELECT id FROM users WHERE id_empresa = ?)`, idQ, idProg, idEmp)
	defer rows.Close()

	distribuicaoCores := make(map[string]int)
	totalRespondentes := 0

	for rows.Next() {
		var userID int
		rows.Scan(&userID)
		pontos := h.TermometroCore(userID, idQ, idProg)

		for _, inter := range intervalos {
			if pontos >= inter.Min && pontos <= inter.Max {
				distribuicaoCores[inter.Cor]++
				break
			}
		}
		totalRespondentes++
	}

	var depSimAnsSim, depSimAnsNao, depNaoAnsSim int

	h.DB.QueryRow(`
		SELECT 
			COUNT(CASE WHEN depressao IS NOT NULL AND ansiedade IS NOT NULL THEN 1 END),
			COUNT(CASE WHEN depressao IS NOT NULL AND ansiedade IS NULL THEN 1 END),
			COUNT(CASE WHEN depressao IS NULL AND ansiedade IS NOT NULL THEN 1 END)
		FROM participantes_questionarios pq
		JOIN users u ON u.id = pq.id_user
		WHERE pq.id_programa = ? AND pq.id_questionario = ? AND u.id_empresa = ?`,
		idProg, idQ, idEmp).Scan(&depSimAnsSim, &depSimAnsNao, &depNaoAnsSim)

	return c.JSON(http.StatusOK, echo.Map{
		"grafico_cores": distribuicaoCores,
		"total":         totalRespondentes,
		"saude_mental": echo.Map{
			"depressao_sim_ansiedade_sim": depSimAnsSim,
			"depressao_sim_ansiedade_nao": depSimAnsNao,
			"depressao_nao_ansiedade_sim": depNaoAnsSim,
		},
	})
}

func (h *EmpresaHandler) RelatorioGraficoCampo(c echo.Context) error {
	idQ := c.Param("id_questionario")
	idEmp := c.Param("id_empresa")
	idProg := c.Param("id_programa")
	idCampo := c.Param("id_campo")

	query := `
		SELECT cer.resposta, u.id
		FROM campos_empresas_respostas cer
		JOIN users u ON cer.id_user = u.id
		WHERE cer.id_empresa = ? AND cer.id_campo = ?
		AND u.id IN (
			SELECT id_user FROM participantes_questionarios 
			WHERE id_questionario = ? AND id_programa = ?
		)`

	rows, _ := h.DB.Query(query, idEmp, idCampo, idQ, idProg)
	defer rows.Close()

	resultadosFiltrados := make(map[string]map[string]int)

	for rows.Next() {
		var valorCampo string
		var userID int
		rows.Scan(&valorCampo, &userID)

		qID, _ := strconv.Atoi(idQ)
		pID, _ := strconv.Atoi(idProg)
		pontos := h.TermometroCore(userID, qID, pID)

		intervalos := h.GetIntervalos(qID)
		var corFinal string
		for _, inter := range intervalos {
			if pontos >= inter.Min && pontos <= inter.Max {
				corFinal = inter.Cor
				break
			}
		}

		if resultadosFiltrados[valorCampo] == nil {
			resultadosFiltrados[valorCampo] = make(map[string]int)
		}
		resultadosFiltrados[valorCampo][corFinal]++
	}

	return c.JSON(http.StatusOK, resultadosFiltrados)
}
