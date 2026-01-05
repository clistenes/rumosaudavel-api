package models

import (
	"time"
)

type Empresa struct {
	ID                        uint           `gorm:"primaryKey;autoIncrement"`
	Nome                      *string        `gorm:"size:255"`
	Introducao                *string        `gorm:"type:text"`
	Cor                       *string        `gorm:"size:255"`
	TermoConsentimento        *string        `gorm:"size:11;column:termo_consentimento"`
	Logotipo                  *string        `gorm:"size:255"`
	Slug                      *string        `gorm:"size:255"`
	IdCampoDashboardHeatmap1  int             `gorm:"column:id_campo_dashboard_heatmap_1;not null"`
	IdCampoDashboardHeatmap2  int             `gorm:"column:id_campo_dashboard_heatmap_2;not null"`
	CreatedAt                 *time.Time
	UpdatedAt                 *time.Time
}

func (Empresa) TableName() string {
	return "empresas"
}
