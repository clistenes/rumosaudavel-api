package handlers

import (
	"github.com/labstack/echo/v4"
	"database/sql"
)

type QuestionarioHandler struct {
	DB *sql.DB
}

func NewQuestionarioHandler(db *sql.DB) *QuestionarioHandler {
	return &QuestionarioHandler{DB: db}
}

func (h *QuestionarioHandler) Criar(c echo.Context) error {
	var b struct {
		Nome, NomeSite, Descricao string
	}
	c.Bind(&b)

	res, _ := h.DB.Exec(`
		INSERT INTO questionarios (nome, nome_site, descricao)
		VALUES (?, ?, ?)
	`, b.Nome, b.NomeSite, b.Descricao)

	id, _ := res.LastInsertId()
	return c.JSON(201, echo.Map{"id": id})
}

func (h *QuestionarioHandler) Listar(c echo.Context) error {
	rows, _ := h.DB.Query(`
		SELECT id, nome, nome_site, descricao
			FROM questionarios
		WHERE deleted IS NULL
		ORDER BY created_at DESC
	`)
	defer rows.Close()

	var out []map[string]interface{}
	for rows.Next() {
		var id int
		var nome, nomeSite, desc string
		rows.Scan(&id, &nome, &nomeSite, &desc)

		out = append(out, echo.Map{
			"id": id, "nome": nome, "nome_site": nomeSite, "descricao": desc,
		})
	}
	return c.JSON(200, out)
}

func (h *QuestionarioHandler) Info(c echo.Context) error {
	id := c.Param("id")

	var q struct {
		ID int
		Nome, NomeSite, Descricao string
	}

	err := h.DB.QueryRow(`
		SELECT id, nome, nome_site, descricao
			FROM questionarios 
		WHERE id=?
	`, id).Scan(&q.ID, &q.Nome, &q.NomeSite, &q.Descricao)

	if err != nil {
		return c.NoContent(404)
	}
	return c.JSON(200, q)
}

func (h *QuestionarioHandler) Editar(c echo.Context) error {
	id := c.Param("id")
	var b struct {
		Nome, NomeSite, Descricao string
	}
	c.Bind(&b)

	h.DB.Exec(`
		UPDATE questionarios
		SET nome = ?, nome_site = ?, descricao = ?
		WHERE id = ?
	`, b.Nome, b.NomeSite, b.Descricao, id)

	return c.NoContent(200)
}

func (h *QuestionarioHandler) Apagar(c echo.Context) error {
	h.DB.Exec(`UPDATE questionarios SET deleted=1 WHERE id=?`, c.Param("id"))
	return c.NoContent(200)
}

func (h *QuestionarioHandler) Duplicar(c echo.Context) error {
	id := c.Param("id")
	tx, _ := h.DB.Begin()

	var nome, nomeSite, desc string
	tx.QueryRow(`
		SELECT nome, nome_site, descricao
		FROM questionarios WHERE id = ?
	`, id).Scan(&nome, &nomeSite, &desc)

	res, _ := tx.Exec(`
		INSERT INTO questionarios (nome, nome_site, descricao)
		VALUES (?, ?, ?)
	`, nome+" cópia", nomeSite, desc)

	newID, _ := res.LastInsertId()

	tx.Exec(`
		INSERT INTO perguntas (id_questionario, nome, nome_site, tipo, pos)
		SELECT ?, nome, nome_site, tipo, pos
		FROM perguntas WHERE id_questionario=?
	`, newID, id)

	tx.Commit()
	return c.JSON(200, echo.Map{"novo_id": newID})
}

func (h *QuestionarioHandler) AddPergunta(c echo.Context) error {
	qid := c.Param("id")
	var b struct {
		Nome, NomeSite, Tipo string
	}
	c.Bind(&b)

	h.DB.Exec(`
		INSERT INTO perguntas (id_questionario, nome, nome_site, tipo)
		VALUES (?, ?, ?, ?)
	`, qid, b.Nome, b.NomeSite, b.Tipo)

	return c.NoContent(201)
}

func (h *QuestionarioHandler) EditarPergunta(c echo.Context) error {
	id := c.Param("id")
	var b struct {
		Nome, NomeSite, Tipo string
	}
	c.Bind(&b)

	h.DB.Exec(`
		UPDATE perguntas
		SET nome=?, nome_site=?, tipo=?
		WHERE id=?
	`, b.Nome, b.NomeSite, b.Tipo, id)

	return c.NoContent(200)
}

func (h *QuestionarioHandler) ApagarPergunta(c echo.Context) error {
	id := c.Param("id")
	h.DB.Exec(`DELETE FROM alternativas WHERE id_pergunta = ?`, id)
	h.DB.Exec(`DELETE FROM perguntas WHERE id = ?`, id)
	return c.NoContent(200)
}

func (h *QuestionarioHandler) AddAlternativa(c echo.Context) error {
	pid := c.Param("id")
	var b struct {
		Texto string
		Pontuacao int
		Tipo, Comentario string
	}
	c.Bind(&b)

	h.DB.Exec(`
		INSERT INTO alternativas (id_pergunta, alternativa, pontuacao, tipo, comentario)
		VALUES (?, ?, ?, ?, ?)
	`, pid, b.Texto, b.Pontuacao, b.Tipo, b.Comentario)

	return c.NoContent(201)
}

func (h *QuestionarioHandler) ApagarAlternativa(c echo.Context) error {
	h.DB.Exec(`DELETE FROM alternativas WHERE id = ?`, c.Param("id"))
	return c.NoContent(200)
}

func (h *QuestionarioHandler) SalvarOrdem(c echo.Context) error {
	var body []struct {
		ID, Position int
	}
	c.Bind(&body)

	for _, p := range body {
		h.DB.Exec(`UPDATE perguntas SET pos = ? WHERE id = ?`, p.Position, p.ID)
	}
	return c.NoContent(200)
}

func (h *QuestionarioHandler) SalvarIntervalo(c echo.Context) error {
	qid := c.Param("id")
	var b struct {
		Inicio, Fim, Texto string
	}
	c.Bind(&b)

	h.DB.Exec(`
		INSERT INTO questionarios_intervalos
		(id_questionario, intervalo_inicio, intervalo_termino, texto)
		VALUES (?, ?, ?, ?)
	`, qid, b.Inicio, b.Fim, b.Texto)

	return c.NoContent(201)
}

func (h *QuestionarioHandler) ApagarIntervalo(c echo.Context) error {
	h.DB.Exec(`DELETE FROM questionarios_intervalos WHERE id = ?`, c.Param("id"))
	return c.NoContent(200)
}

func (h *QuestionarioHandler) Restaurar(c echo.Context) error {
	h.DB.Exec(`UPDATE questionarios SET deleted = NULL WHERE id = ?`, c.Param("id"))
	return c.NoContent(200)
}

func (h *QuestionarioHandler) DeleteDefinitivo(c echo.Context) error {
	h.DB.Exec(`UPDATE questionarios SET deleted = 2 WHERE id = ?`, c.Param("id"))
	return c.NoContent(200)
}

