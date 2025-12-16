package models

import (
	"time"

	"gorm.io/gorm"
)

type ModeloBase struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"index"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	CreatedBy *uint `gorm:"default:null"`
	UpdatedBy *uint `gorm:"default:null"`
	DeletedBy *uint `gorm:"default:null"`
}
