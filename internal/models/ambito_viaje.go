package models

import "time"

type AmbitoViaje struct {
	Codigo    string    `gorm:"primaryKey;size:20;not null" json:"codigo"`
	Nombre    string    `gorm:"size:50;not null;unique" json:"nombre"`
	CreatedAt time.Time `gorm:"type:timestamp" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp" json:"updated_at"`
}

func (AmbitoViaje) TableName() string { return "ambito_viajes" }
