package models

import "time"

type PasswordReset struct {
	Email     string     `gorm:"index;size:255"`
	Token     string     `gorm:"size:255"`
	CreatedAt *time.Time `gorm:"column:created_at"`
}

func (PasswordReset) TableName() string {
	return "password_resets"
}