package models

import (
	"time"

	"gorm.io/gorm"
)

type EstadoVoucher struct {
	Codigo      string `gorm:"primaryKey;size:50;not null"`
	Nombre      string `gorm:"size:100;not null"`
	Descripcion string `gorm:"size:255"`
	Color       string `gorm:"size:20;default:'gray'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (EstadoVoucher) TableName() string {
	return "estados_voucher"
}
