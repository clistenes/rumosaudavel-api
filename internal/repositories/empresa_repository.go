package repositories

import (
	"rumosaudavel-api/internal/models"

	"gorm.io/gorm"
)

type EmpresaRepository struct {
	DB *gorm.DB
}

func (r *EmpresaRepository) Create(empresa *models.Empresa) error {
	return r.DB.Create(empresa).Error
}

func (r *EmpresaRepository) FindAll() ([]models.Empresa, error) {
	var empresas []models.Empresa
	err := r.DB.Find(&empresas).Error
	return empresas, err
}

func (r *EmpresaRepository) FindByID(id uint) (*models.Empresa, error) {
	var empresa models.Empresa
	err := r.DB.First(&empresa, id).Error
	return &empresa, err
}

func (r *EmpresaRepository) FindBySlug(slug string) (*models.Empresa, error) {
	var empresa models.Empresa
	err := r.DB.Where("slug = ?", slug).First(&empresa).Error
	return &empresa, err
}

func (r *EmpresaRepository) Update(empresa *models.Empresa) error {
	return r.DB.Save(empresa).Error
}
