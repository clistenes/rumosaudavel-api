package handlers

import (
	"database/sql"
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

func (h *ProgramaHandler) Criar(c echo.Context) error {
	type Request struct {
		Nome          string `json:"nome"`
		Introducao    string `json:"introducao"`
		Questionarios []int  `json:"questionarios"`
	}

	var req Request
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, echo.Map{"error": "dados inválidos"})
	}

	var exists int
	h.DB.QueryRow(`SELECT COUNT(*) FROM programas WHERE nome = ?`, req.Nome).Scan(&exists)
	if exists > 0 {
		return c.JSON(409, echo.Map{"error": "nome do programa já existe"})
	}

	res, err := h.DB.Exec(`
		INSERT INTO programas (nome, introducao)
		VALUES (?, ?)
	`, req.Nome, req.Introducao)
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao criar programa"})
	}

	programaID, _ := res.LastInsertId()

	for _, q := range req.Questionarios {
		h.DB.Exec(`
			INSERT INTO programas_questionarios (id_programa, id_questionario)
			VALUES (?, ?)
		`, programaID, q)
	}

	return c.JSON(201, echo.Map{
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
	c.Bind(&req)

	_, err := h.DB.Exec(`
		UPDATE programas
			SET nome = ?, introducao = ?, ordenacao_questionarios = ?
		WHERE id = ?
	`, req.Nome, req.Introducao, req.Ordenacao, req.ID)

	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao atualizar programa"})
	}

	h.DB.Exec(`DELETE FROM programas_questionarios WHERE id_programa = ?`, req.ID)

	for _, q := range req.Questionarios {
		h.DB.Exec(`
			INSERT INTO programas_questionarios (id_programa, id_questionario)
			VALUES (?, ?)
		`, req.ID, q)
	}

	return c.JSON(200, echo.Map{"status": "atualizado"})
}

func (h *ProgramaHandler) Lista(c echo.Context) error {
	rows, err := h.DB.Query(`
		SELECT id, nome, introducao, created_at
		FROM programas
		ORDER BY created_at DESC
	`)
	if err != nil {
		return c.JSON(500, echo.Map{"error": "erro ao buscar programas"})
	}
	defer rows.Close()

	type Programa struct {
		ID         int       `json:"id"`
		Nome       string    `json:"nome"`
		Introducao string    `json:"introducao"`
		CreatedAt  time.Time `json:"created_at"`
	}

	var programas []Programa

	for rows.Next() {
		var p Programa
		rows.Scan(&p.ID, &p.Nome, &p.Introducao, &p.CreatedAt)
		programas = append(programas, p)
	}

	return c.JSON(200, programas)
}

func (h *ProgramaHandler) VincularEmpresa(c echo.Context) error {
	type Request struct {
		EmpresaID  int `json:"id_empresa"`
		ProgramaID int `json:"id_programa"`
	}

	var req Request
	c.Bind(&req)

	var count int
	h.DB.QueryRow(`
		SELECT COUNT(*)
		FROM programas_empresas
		WHERE id_empresa = ? AND id_programa = ?
	`, req.EmpresaID, req.ProgramaID).Scan(&count)

	if count == 0 {
		h.DB.Exec(`
			INSERT INTO programas_empresas
			(id_empresa, id_programa, intervalo_tipo)
			VALUES (?, ?, 'indeterminado')
		`, req.EmpresaID, req.ProgramaID)
	}

	return c.JSON(200, echo.Map{"status": "vinculado"})
}

func (h *ProgramaHandler) DefinirIntervalo(c echo.Context) error {
	type Request struct {
		EmpresaID  int    `json:"id_empresa"`
		ProgramaID int    `json:"id_programa"`
		Inicio     string `json:"inicio"`
		Termino    string `json:"termino"`
	}

	var req Request
	c.Bind(&req)

	inicio := strings.ReplaceAll(req.Inicio, "/", "-")
	termino := strings.ReplaceAll(req.Termino, "/", "-")

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
	empresaID := c.Param("empresa")
	programaID := c.Param("programa")

	h.DB.Exec(`
		UPDATE programas_empresas
			SET intervalo_inicio = NULL, intervalo_termino = NULL
		WHERE id_empresa = ? AND id_programa = ?
	`, empresaID, programaID)

	return c.JSON(200, echo.Map{"status": "intervalo removido"})
}

func (h *ProgramaHandler) Apagar(c echo.Context) error {
	id := c.Param("id")

	h.DB.Exec(`DELETE FROM programas_questionarios WHERE id_programa = ?`, id)
	h.DB.Exec(`DELETE FROM programas_empresas WHERE id_programa = ?`, id)
	h.DB.Exec(`DELETE FROM programas WHERE id = ?`, id)

	return c.NoContent(204)
}

func (h *ProgramaHandler) Duplicar(c echo.Context) error {
	id := c.Param("id")

	tx, _ := h.DB.Begin()

	var nome, introducao string
	tx.QueryRow(`
		SELECT nome, introducao
		FROM programas
		WHERE id = ?
	`, id).Scan(&nome, &introducao)

	res, _ := tx.Exec(`
		INSERT INTO programas (nome, introducao)
		VALUES (?, ?)
	`, nome+" cópia", introducao)

	novoID, _ := res.LastInsertId()

	tx.Exec(`
		INSERT INTO programas_empresas (id_programa, id_empresa, intervalo_tipo)
		SELECT ?, id_empresa, intervalo_tipo
		FROM programas_empresas
		WHERE id_programa = ?
	`, novoID, id)

	tx.Exec(`
		INSERT INTO programas_questionarios (id_programa, id_questionario)
		SELECT ?, id_questionario
		FROM programas_questionarios
		WHERE id_programa = ?
	`, novoID, id)

	tx.Commit()

	return c.JSON(201, echo.Map{
		"id_programa": novoID,
		"status":     "duplicado",
	})
}
