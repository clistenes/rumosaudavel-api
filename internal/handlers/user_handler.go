package handlers

import (
	"fmt"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"net/http"
	"time"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	DB *sql.DB
}

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

type JwtCustomClaims struct {
	UserID    int    `json:"user_id"`
	Type      string `json:"type"`
	EmpresaID *int   `json:"empresa_id"`
	jwt.RegisteredClaims
}

func NewUserHandler(db *sql.DB) *UserHandler {
	return &UserHandler{DB: db}
}

func (h *UserHandler) Criar(c echo.Context) error {
	id := c.FormValue("id_usuario")
	login := c.FormValue("usuario_login")
	nome := c.FormValue("usuario_nome")
	email := c.FormValue("usuario_email")
	senha := c.FormValue("usuario_senha")

	if nome == "" || login == "" || email == "" {
		return c.JSON(http.StatusBadRequest, "nome, login e email obrigatórios")
	}

	var count int
	query := `SELECT COUNT(*) FROM users WHERE login = ?`
	args := []any{login}

	if id != "" {
		query += ` AND id != ?`
		args = append(args, id)
	}

	h.DB.QueryRow(query, args...).Scan(&count)
	if count > 0 {
		return c.JSON(http.StatusBadRequest, "login já existe")
	}

	if id == "" && senha == "" {
		return c.JSON(http.StatusBadRequest, "senha obrigatória")
	}

	if id == "" {
		hash, _ := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
		_, err := h.DB.Exec(`
			INSERT INTO users (name, login, email, password, type)
			VALUES (?, ?, ?, ?, '1')
		`, nome, login, email, hash)
		if err != nil {
			return err
		}
	} else {
		if senha != "" {
			hash, _ := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
			_, _ = h.DB.Exec(`
				UPDATE users SET name=?, login=?, password=? WHERE id=?
			`, nome, email, hash, id)
		} else {
			_, _ = h.DB.Exec(`
				UPDATE users SET name=?, login=? WHERE id=?
			`, nome, email, id)
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) Lista(c echo.Context) error {
	rows, _ := h.DB.Query(`
		SELECT id, name, login FROM users WHERE type='1'
	`)
	defer rows.Close()

	var usuarios []echo.Map
	for rows.Next() {
		var id int
		var nome, login string
		rows.Scan(&id, &nome, &login)
		usuarios = append(usuarios, echo.Map{
			"id": id, "nome": nome, "login": login,
		})
	}

	return c.JSON(http.StatusOK, usuarios)
}

func (h *UserHandler) Apagar(c echo.Context) error {
	id := c.Param("id")
	_, _ = h.DB.Exec(`DELETE FROM users WHERE id=?`, id)
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) Login(c echo.Context) error {
	login := c.FormValue("login")
	password := c.FormValue("password")

	var (
		id        int
		hash      string
		tipo      string
		empresaID sql.NullInt64
	)

	err := h.DB.QueryRow(`
		SELECT id, password, type, id_empresa
		FROM users WHERE login=?
	`, login).Scan(&id, &hash, &tipo, &empresaID)

	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return c.JSON(http.StatusUnauthorized, "login ou senha incorretos")
	}

	claims := JwtCustomClaims{
		UserID: id,
		Type:   tipo,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	if empresaID.Valid {
		eid := int(empresaID.Int64)
		claims.EmpresaID = &eid
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t, _ := token.SignedString(jwtSecret)

	return c.JSON(http.StatusOK, echo.Map{
		"token": t,
		"type":  tipo,
		"empresa_id": claims.EmpresaID,
	})
}

func (h *UserHandler) Logout(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func randomToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *UserHandler) Esqueci(c echo.Context) error {
	email := c.FormValue("email")

	var id int
	err := h.DB.QueryRow(`
		SELECT id FROM users WHERE email=?
	`, email).Scan(&id)

	if err != nil {
		return c.JSON(http.StatusBadRequest, "email não encontrado")
	}

	token := randomToken() + "-" + fmt.Sprint(id)

	_, _ = h.DB.Exec(`
		UPDATE users SET remember_token=? WHERE id=?
	`, token, id)

	// enviar email com o token (inserido no resetSenha)
	fmt.Printf("Token de reset de senha para %s: %s\n", email, token)
	
	return c.NoContent(http.StatusOK)
}

func (h *UserHandler) ResetSenha(c echo.Context) error {
	token := c.Param("token")
	novaSenha := c.FormValue("password")

	var id int
	err := h.DB.QueryRow(`
		SELECT id FROM users WHERE remember_token=?
	`, token).Scan(&id)

	if err != nil {
		return echo.ErrBadRequest
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)

	_, _ = h.DB.Exec(`
		UPDATE users SET password=?, remember_token='' WHERE id=?
	`, hash, id)

	return c.NoContent(http.StatusOK)
}
