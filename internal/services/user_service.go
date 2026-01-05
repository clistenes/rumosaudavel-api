package services

import (
	"errors"
	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/repositories"
	"rumosaudavel-api/internal/utils"
)

type UserService struct {
	Repo *repositories.UserRepository
}

func (s *UserService) Create(name, email, password string) error {
	hash, err := utils.HashPassword(password)
	if err != nil {
		return err
	}

	user := models.User{
		Name:     name,
		Email:    email,
		Password: hash,
	}

	return s.Repo.Create(&user)
}

func (s *UserService) List() ([]models.User, error) {
	return s.Repo.FindAll()
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	return s.Repo.FindByID(id)
}

func (s *UserService) Update(id uint, name, email, password string) error {
	user, err := s.Repo.FindByID(id)
	if err != nil {
		return errors.New("usuário não encontrado")
	}

	user.Name = name
	user.Email = email

	if password != "" {
		hash, _ := utils.HashPassword(password)
		user.Password = hash
	}

	return s.Repo.Update(user)
}
