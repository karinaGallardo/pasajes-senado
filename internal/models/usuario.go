package models

import (
	"time"

	"gorm.io/gorm"
)

type Usuario struct {
	ID        string    `gorm:"primaryKey;size:24;not null"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	CI       string `gorm:"size:20;index"`
	Username string `gorm:"uniqueIndex;size:100;not null"`
}

func (Usuario) TableName() string {
	return "usuarios"
}
