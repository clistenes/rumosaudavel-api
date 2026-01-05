package handlers

import (
	"net/http"

	"rumosaudavel-api/internal/services"
	"rumosaudavel-api/internal/tokens"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	AuthService *services.AuthService
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "JSON inválido"})
	}

	err := h.AuthService.Register(req.Name, req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, echo.Map{"message": "Usuário criado com sucesso"})
}

func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "JSON inválido"})
	}

	user, err := h.AuthService.Login(req.Email, req.Password)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, echo.Map{"error": err.Error()})
	}

	token, _ := tokens.GenerateJWT(user.ID)
	return c.JSON(http.StatusOK, echo.Map{"token": token})
}

func (h *AuthHandler) Me(c echo.Context) error {
	user := c.Get("userData").(map[string]interface{})
	return c.JSON(http.StatusOK, echo.Map{
		"user_id": user["user_id"],
	})
}
