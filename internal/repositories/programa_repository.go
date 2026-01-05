package repositories

import (
	"rumosaudavel-api/internal/models"

	"gorm.io/gorm"
)

type ProgramaRepository struct {
	DB *gorm.DB
}

func (r *ProgramaRepository) Create(programa *models.Programa) error {
	return r.DB.Create(programa).Error
}

func (r *ProgramaRepository) FindAll() ([]models.Programa, error) {
	var programas []models.Programa
	err := r.DB.Find(&programas).Error
	return programas, err
}

func (r *ProgramaRepository) FindByID(id uint) (*models.Programa, error) {
	var programa models.Programa
	err := r.DB.First(&programa, id).Error
	return &programa, err
}

func (r *ProgramaRepository) Update(programa *models.Programa) error {
	return r.DB.Save(programa).Error
}

