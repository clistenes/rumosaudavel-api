package handlers

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

type DashboardHandler struct {
	DB *sql.DB
}

func NewDashboardHandler(db *sql.DB) *DashboardHandler {
	return &DashboardHandler{DB: db}
}

func (h *DashboardHandler) Home(c echo.Context) error {
	var participantes int
	var empresas int
	var questionarios int

	err := h.DB.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM users WHERE id_empresa IS NOT NULL AND type = ?) AS participantes,
			(SELECT COUNT(*) FROM empresas) AS empresas,
			(SELECT COUNT(*) FROM questionarios WHERE deleted IS NULL) AS questionarios
	`, 2).Scan(&participantes, &empresas, &questionarios)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "erro ao buscar dados do dashboard",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"participantes": participantes,
		"empresas":       empresas,
		"questionarios":  questionarios,
	})
}

type LogError struct {
	ID        int    `json:"id"`
	Message   string `json:"message"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	CreatedAt string `json:"created_at"`
}

func (h *DashboardHandler) Bugs(c echo.Context) error {
	rows, err := h.DB.Query(`
		SELECT id, message, file, line, created_at
		FROM log_errors
		ORDER BY created_at DESC
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "erro ao buscar logs",
		})
	}
	defer rows.Close()

	var logs []LogError

	for rows.Next() {
		var log LogError
		if err := rows.Scan(
			&log.ID,
			&log.Message,
			&log.File,
			&log.Line,
			&log.CreatedAt,
		); err != nil {
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"error": "erro ao mapear logs",
			})
		}
		logs = append(logs, log)
	}

	return c.JSON(http.StatusOK, echo.Map{
		"erros": logs,
	})
}
