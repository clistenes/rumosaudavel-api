package handlers

import (
	"net/http"
	"strconv"

	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/services"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	Service *services.EmpresaService
}

func (h *Handler) Create(c echo.Context) error {
	var empresa models.Empresa

	if err := c.Bind(&empresa); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	if err := h.Service.Create(&empresa); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, empresa)
}

func (h *Handler) List(c echo.Context) error {
	empresas, err := h.Service.List()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, empresas)
}

func (h *Handler) Get(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	empresa, err := h.Service.Get(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, empresa)
}

func (h *Handler) Update(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var payload models.Empresa
	if err := c.Bind(&payload); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	if err := h.Service.Update(uint(id), &payload); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "empresa atualizada"})
}

