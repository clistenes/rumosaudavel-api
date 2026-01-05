package handlers

import (
	"net/http"
	"strconv"

	"rumosaudavel-api/internal/services"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	Service *services.UserService
}

type UserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Create(c echo.Context) error {
	var req UserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "dados inválidos"})
	}

	err := h.Service.Create(req.Name, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "usuário criado"})
}

func (h *UserHandler) List(c echo.Context) error {
	users, _ := h.Service.List()
	return c.JSON(http.StatusOK, users)
}

func (h *UserHandler) Get(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	user, err := h.Service.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "usuário não encontrado"})
	}
	return c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Update(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))
	var req UserRequest
	c.Bind(&req)

	err := h.Service.Update(uint(id), req.Name, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "usuário atualizado"})
}
