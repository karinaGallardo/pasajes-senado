package models

import "time"

type Ciudad struct {
	Code      string `gorm:"primaryKey;size:4;not null"`
	Nombre    string `gorm:"column:nombre;size:100;not null;unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Ciudad) TableName() string {
	return "ciudades"
}
