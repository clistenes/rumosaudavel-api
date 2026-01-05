package handlers

import (
	"net/http"
	"strconv"

	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/services"

	"github.com/labstack/echo/v4"
)

type ProgramaHandler struct {
	Service *services.ProgramaService
}

func (h *ProgramaHandler) Create(c echo.Context) error {
	var programa models.Programa

	if err := c.Bind(&programa); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	if err := h.Service.Create(&programa); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, programa)
}

func (h *ProgramaHandler) List(c echo.Context) error {
	programas, err := h.Service.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, programas)
}

func (h *ProgramaHandler) Get(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	programa, err := h.Service.Get(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, programa)
}

func (h *ProgramaHandler) Update(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var payload models.Programa
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	if err := h.Service.Update(uint(id), &payload); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "programa atualizado"})
}
