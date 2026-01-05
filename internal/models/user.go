package models

import (
	"time"
)

type User struct {
	ID              uint       `gorm:"primaryKey;autoIncrement"`
	Name            string     `gorm:"size:255;not null"`
	Email           string     `gorm:"size:255;uniqueIndex;not null"`
	EmailVerifiedAt *time.Time `gorm:"column:email_verified_at"`
	Password        string     `gorm:"size:255;not null"`
	RememberToken   *string    `gorm:"column:remember_token"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (User) TableName() string {
	return "users"
}
