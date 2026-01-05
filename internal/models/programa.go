package models

import (
	"time"
)

type Programa struct {
	ID                    uint           `gorm:"primaryKey;autoIncrement"`
	Nome                  *string        `gorm:"type:text"`
	Introducao            *string        `gorm:"type:text"`
	OrdenacaoQuestionarios string        `gorm:"size:100;not null;column:ordenacao_questionarios"`
	CreatedAt             *time.Time
	UpdatedAt             *time.Time
}

func (Programa) TableName() string {
	return "programas"
}
