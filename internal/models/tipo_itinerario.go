package models

import "time"

type TipoItinerario struct {
	Codigo    string    `gorm:"primaryKey;size:20;not null" json:"codigo"`
	Nombre    string    `gorm:"size:50;not null;unique" json:"nombre"`
	CreatedAt time.Time `gorm:"type:timestamp"`
	UpdatedAt time.Time `gorm:"type:timestamp"`
}

func (TipoItinerario) TableName() string { return "tipo_itinerarios" }
