package models

import (
	"time"

	"gorm.io/gorm"
)

type EstadoSolicitud struct {
	Codigo      string `gorm:"primaryKey;size:30;not null"`
	Nombre      string `gorm:"size:100;not null"`
	Descripcion string `gorm:"size:255"`
	Color       string `gorm:"size:20;default:'gray'"`

	CreatedAt time.Time      `gorm:"type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`
}

func (EstadoSolicitud) TableName() string {
	return "estados_solicitud"
}
