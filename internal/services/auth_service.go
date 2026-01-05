package services

import (
	"errors"
	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/repositories"
	"rumosaudavel-api/internal/utils"
)

type AuthService struct {
	UserRepo *repositories.UserRepository
}

func (s *AuthService) Register(name, email, password string) error {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}
	user := models.User{Name: name, Email: email, Password: hash}
	return s.UserRepo.Create(&user)
}

func (s *AuthService) Login(email, password string) (*models.User, error) {
	user, err := s.UserRepo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("usuário não encontrado")
	}
	if !utils.CheckPassword(password, user.Password) {
		return nil, errors.New("senha incorreta")
	}
	return user, nil
}