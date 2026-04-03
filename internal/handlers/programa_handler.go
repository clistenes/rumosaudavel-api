package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

type ProgramaHandler struct {
	DB *sql.DB
}

func NewProgramaHandler(db *sql.DB) *ProgramaHandler {
	return &ProgramaHandler{DB: db}
}

type Programa struct {
	ID         int       `json:"id"`
	Nome       string    `json:"nome"`
	Introducao string    `json:"introducao"`
	CreatedAt  time.Time `json:"created_at"`
}

func (h *ProgramaHandler) Criar(c echo.Context) error {
	type Request struct {
		Nome          string `json:"nome"`
		Introducao    string `json:"introducao"`
		Ordenacao     string `json:"ordenacao_questionarios"`
		Questionarios []int  `json:"questionarios"`
	}

	var body map[string]interface{}

	if err := c.Bind(&body); err != nil {
		return c.JSON(400, echo.Map{"error": "dados inválidos"})
	}

	if body["nome"] == nil || body["introducao"] == nil || body["ordenacao_questionarios"] == nil {
		return c.JSON(400, echo.Map{"error": "campos obrigatórios: nome, introducao, ordenacao_questionarios"})
	}

	req := Request{
		Nome:       body["nome"].(string),
		Introducao: body["introducao"].(string),
		Ordenacao:  body["ordenacao_questionarios"].(string),
	}

	if strings.TrimSpace(req.Nome) == "" {
		return c.JSON(400, echo.Map{"error": "nome é obrigatório"})
	}

	var exists int
	err := h.DB.QueryRow(
		`SELECT COUNT(*) FROM programas WHERE nome = ?`,
		req.Nome,
	).Scan(&exists)

	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao validar programa"})
	}

	if exists > 0 {
		return c.JSON(409, echo.Map{"error": "nome do programa já existe"})
	}

	tx, err := h.DB.Begin()
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao iniciar transação"})
	}

	res, err := tx.Exec(`
		INSERT INTO programas (created_at, updated_at, nome, introducao, ordenacao_questionarios)
		VALUES (?, ?, ?, ?, ?)
	`, time.Now(), time.Now(), req.Nome, req.Introducao, req.Ordenacao)

	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao criar programa"})
	}

	programaID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao obter id"})
	}

	for _, q := range req.Questionarios {

		_, err := tx.Exec(`
			INSERT INTO programas_questionarios
			(id_programa, id_questionario)
			VALUES (?, ?)
		`, programaID, q)

		if err != nil {
			tx.Rollback()
			return c.JSON(500, echo.Map{"error": "erro ao vincular questionário"})
		}
	}

	tx.Commit()

	return c.JSON(http.StatusCreated, echo.Map{
		"id_programa": programaID,
		"status":      "criado",
	})
}

func (h *ProgramaHandler) Editar(c echo.Context) error {
	type Request struct {
		ID            int    `json:"id"`
		Nome          string `json:"nome"`
		Introducao    string `json:"introducao"`
		Ordenacao     string `json:"ordenacao_questionarios"`
		Questionarios []int  `json:"questionarios"`
	}

	var req Request

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.Map{"error": "dados inválidos"})
	}

	fmt.Printf("Dados recebidos para edição: %+v\n", req)

	tx, err := h.DB.Begin()
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao iniciar transação"})
	}

	count := 0
	err = tx.QueryRow(`SELECT COUNT(*) FROM programas WHERE id = ?`, req.ID).Scan(&count)

	if count == 0 {
		tx.Rollback()
		return c.JSON(404, echo.Map{"error": "programa não encontrado"})
	}

	fmt.Printf("Programa encontrado com ID: %d\n", req.ID)

	_, err = tx.Exec(`
		UPDATE programas
		SET nome = ?, introducao = ?, ordenacao_questionarios = ?, updated_at = ?
		WHERE id = ?
	`, req.Nome, req.Introducao, req.Ordenacao, time.Now(), req.ID)

	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao atualizar programa"})
	}

	_, err = tx.Exec(`DELETE FROM programas_questionarios WHERE id_programa = ?`, req.ID)
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao atualizar questionários"})
	}

	for _, q := range req.Questionarios {
		_, err := tx.Exec(`
			INSERT INTO programas_questionarios
			(id_programa, id_questionario)
			VALUES (?, ?)
		`, req.ID, q)

		if err != nil {
			tx.Rollback()
			return c.JSON(500, echo.Map{"error": "erro ao inserir questionário"})
		}
	}

	tx.Commit()

	return c.JSON(200, echo.Map{"status": "atualizado"})
}

func (h *ProgramaHandler) Lista(c echo.Context) error {
	rows, err := h.DB.Query(`
		SELECT id, COALESCE(nome, '') as nome, COALESCE(introducao, '') as introducao, created_at
			FROM programas
		ORDER BY created_at DESC
	`)
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao buscar programas"})
	}
	defer rows.Close()

	var programas []Programa

	for rows.Next() {
		var p Programa

		err := rows.Scan(&p.ID, &p.Nome, &p.Introducao, &p.CreatedAt)
		if err != nil {
			return c.JSON(500, echo.Map{"error": "erro ao processar resultado"})
		}

		programas = append(programas, p)
	}

	return c.JSON(200, programas)
}

func (h *ProgramaHandler) Empresas(c echo.Context) error {
	programaID := c.QueryParam("id")
	if programaID == "" {
		return c.JSON(400, echo.Map{"error": "id do programa é obrigatório"})
	}

	Id, err := strconv.Atoi(programaID)
	if err != nil {
		return c.JSON(400, echo.Map{"error": "id inválido"})
	}

	rows, err := h.DB.Query(`
		SELECT e.id, e.nome, e.logotipo, pe.intervalo_inicio, pe.intervalo_termino
			FROM empresas e
		INNER JOIN programas_empresas pe ON (e.id = pe.id_empresa)
		WHERE pe.id_programa = ?
	`, Id)
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao buscar empresas"})
	}
	defer rows.Close()

	type EmpresaVinculada struct {
		ID               int     `json:"id"`
		Nome             string  `json:"nome"`
		Logotipo         *string `json:"logotipo,omitempty"`
		IntervaloInicio  *string `json:"intervalo_inicio,omitempty"`
		IntervaloTermino *string `json:"intervalo_termino,omitempty"`
	}

	var empresas []EmpresaVinculada

	for rows.Next() {
		var ev EmpresaVinculada
		var logotipo sql.NullString
		var inicio, termino sql.NullString

		err := rows.Scan(&ev.ID, &ev.Nome, &logotipo, &inicio, &termino)
		if err != nil {
			return c.JSON(500, echo.Map{"error": "erro ao processar resultado"})
		}

		if logotipo.Valid {
			ev.Logotipo = &logotipo.String
		}

		if inicio.Valid {
			ev.IntervaloInicio = &inicio.String
		}

		if termino.Valid {
			ev.IntervaloTermino = &termino.String
		}

		empresas = append(empresas, ev)
	}

	if len(empresas) == 0 {
		return c.JSON(404, echo.Map{"error": "nenhuma empresa encontrada para o programa especificado"})
	}

	return c.JSON(200, empresas)
}

func (h *ProgramaHandler) VincularEmpresa(c echo.Context) error {
	type Request struct {
		EmpresaID  int `json:"id_empresa"`
		ProgramaID int `json:"id_programa"`
	}

	var req Request

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.Map{"error": "dados inválidos"})
	}

	var count int

	err := h.DB.QueryRow(`
		SELECT COUNT(*)
		FROM programas_empresas
		WHERE id_empresa = ? AND id_programa = ?
	`, req.EmpresaID, req.ProgramaID).Scan(&count)

	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao verificar vínculo"})
	}

	if count == 0 {
		_, err := h.DB.Exec(`
			INSERT INTO programas_empresas
			(id_programa, id_empresa, intervalo_tipo)
			VALUES (?, ?, 'indeterminado')
		`, req.ProgramaID, req.EmpresaID)

		if err != nil {
			return c.JSON(500, echo.Map{"error": "erro ao vincular"})
		}
	}

	return c.JSON(200, echo.Map{"status": "vinculado"})
}

func parseDateBR(date string) string {

	date = strings.ReplaceAll(date, "/", "-")

	t, err := time.Parse("02-01-2006", date)
	if err != nil {
		return ""
	}

	return t.Format("2006-01-02")
}

func (h *ProgramaHandler) DefinirIntervalo(c echo.Context) error {
	type Request struct {
		EmpresaID  int    `json:"id_empresa"`
		ProgramaID int    `json:"id_programa"`
		Inicio     string `json:"inicio"`
		Termino    string `json:"termino"`
	}

	var req Request

	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.Map{"error": "dados inválidos"})
	}

	inicio := parseDateBR(req.Inicio)
	termino := parseDateBR(req.Termino)

	_, err := h.DB.Exec(`
		UPDATE programas_empresas
		SET intervalo_inicio = ?, intervalo_termino = ?
		WHERE id_empresa = ? AND id_programa = ?
	`, inicio, termino, req.EmpresaID, req.ProgramaID)

	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao definir intervalo"})
	}

	return c.JSON(200, echo.Map{"status": "intervalo atualizado"})
}

func (h *ProgramaHandler) ResetarIntervalo(c echo.Context) error {
	empresaID, err := strconv.Atoi(c.Param("empresa"))
	if err != nil {
		return c.JSON(400, echo.Map{"error": "empresa inválida"})
	}

	programaID, err := strconv.Atoi(c.Param("programa"))
	if err != nil {
		return c.JSON(400, echo.Map{"error": "programa inválido"})
	}

	_, err = h.DB.Exec(`
		UPDATE programas_empresas
		SET intervalo_inicio = NULL,
		    intervalo_termino = NULL
		WHERE id_empresa = ? AND id_programa = ?
	`, empresaID, programaID)

	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao resetar intervalo"})
	}

	return c.JSON(200, echo.Map{"status": "intervalo removido"})
}

func (h *ProgramaHandler) Apagar(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(400, echo.Map{"error": "id inválido"})
	}

	tx, err := h.DB.Begin()
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao iniciar transação"})
	}

	_, err = tx.Exec(`DELETE FROM programas_questionarios WHERE id_programa = ?`, id)
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao remover questionários"})
	}

	_, err = tx.Exec(`DELETE FROM programas_empresas WHERE id_programa = ?`, id)
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao remover vínculos"})
	}

	_, err = tx.Exec(`DELETE FROM programas WHERE id = ?`, id)
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao remover programa"})
	}

	tx.Commit()

	return c.JSON(200, echo.Map{"status": "removido"})
}

func (h *ProgramaHandler) Duplicar(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(400, echo.Map{"error": "id inválido"})
	}

	tx, err := h.DB.Begin()
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao iniciar transação"})
	}

	var nome, introducao, ordenacaoQuestionarios string

	err = tx.QueryRow(`
		SELECT nome, introducao, ordenacao_questionarios
		FROM programas
		WHERE id = ?
	`, id).Scan(&nome, &introducao, &ordenacaoQuestionarios)

	if err != nil {
		tx.Rollback()
		return c.JSON(404, echo.Map{"error": "programa não encontrado"})
	}

	res, err := tx.Exec(`
		INSERT INTO programas (nome, introducao, ordenacao_questionarios, created_at, updated_at)
		VALUES (?, ?, ?, NOW(), NOW())
	`, nome+" cópia", introducao, ordenacaoQuestionarios)

	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao duplicar programa"})
	}

	novoID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao obter id"})
	}

	_, err = tx.Exec(`
		INSERT INTO programas_empresas (id_programa, id_empresa, intervalo_tipo)
		SELECT ?, id_empresa, intervalo_tipo
		FROM programas_empresas
		WHERE id_programa = ?
	`, novoID, id)

	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao copiar empresas"})
	}

	_, err = tx.Exec(`
		INSERT INTO programas_questionarios (id_programa, id_questionario)
		SELECT ?, id_questionario
		FROM programas_questionarios
		WHERE id_programa = ?
	`, novoID, id)

	if err != nil {
		tx.Rollback()
		return c.JSON(500, echo.Map{"error": "erro ao copiar questionários"})
	}

	tx.Commit()

	return c.JSON(201, echo.Map{
		"id_programa": novoID,
		"status":      "duplicado",
	})
}
