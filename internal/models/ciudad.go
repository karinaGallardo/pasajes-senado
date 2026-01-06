package models

import "time"

type Ciudad struct {
	Code      string    `gorm:"primaryKey;size:4;not null"`
	Nombre    string    `gorm:"column:nombre;size:100;not null;unique"`
	CreatedAt time.Time `gorm:"type:timestamp"`
	UpdatedAt time.Time `gorm:"type:timestamp"`
}

func (Ciudad) TableName() string {
	return "ciudades"
}
