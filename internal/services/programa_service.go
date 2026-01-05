package services

import (
	"errors"
	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/repositories"
)

type ProgramaService struct {
	Repo *repositories.ProgramaRepository
}

func (s *ProgramaService) Create(p *models.Programa) error {
	return s.Repo.Create(p)
}

func (s *ProgramaService) List() ([]models.Programa, error) {
	return s.Repo.FindAll()
}

func (s *ProgramaService) Get(id uint) (*models.Programa, error) {
	return s.Repo.FindByID(id)
}

func (s *ProgramaService) Update(id uint, data *models.Programa) error {
	programa, err := s.Repo.FindByID(id)
	if err != nil {
		return errors.New("programa não encontrado")
	}

	programa.Nome = data.Nome
	programa.Introducao = data.Introducao
	programa.OrdenacaoQuestionarios = data.OrdenacaoQuestionarios

	return s.Repo.Update(programa)
}

