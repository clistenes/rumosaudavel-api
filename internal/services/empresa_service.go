package services

import (
	"errors"
	"rumosaudavel-api/internal/models"
	"rumosaudavel-api/internal/repositories"
)

type EmpresaService struct {
	Repo *repositories.EmpresaRepository
}

func (s *EmpresaService) Create(e *models.Empresa) error {
	return s.Repo.Create(e)
}

func (s *EmpresaService) List() ([]models.Empresa, error) {
	return s.Repo.FindAll()
}

func (s *EmpresaService) Get(id uint) (*models.Empresa, error) {
	return s.Repo.FindByID(id)
}

func (s *EmpresaService) GetBySlug(slug string) (*models.Empresa, error) {
	return s.Repo.FindBySlug(slug)
}

func (s *EmpresaService) Update(id uint, data *models.Empresa) error {
	empresa, err := s.Repo.FindByID(id)
	if err != nil {
		return errors.New("empresa não encontrada")
	}

	empresa.Nome = data.Nome
	empresa.Introducao = data.Introducao
	empresa.Cor = data.Cor
	empresa.TermoConsentimento = data.TermoConsentimento
	empresa.Logotipo = data.Logotipo
	empresa.Slug = data.Slug
	empresa.IdCampoDashboardHeatmap1 = data.IdCampoDashboardHeatmap1
	empresa.IdCampoDashboardHeatmap2 = data.IdCampoDashboardHeatmap2

	return s.Repo.Update(empresa)
}