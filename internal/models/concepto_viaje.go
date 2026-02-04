package models

import "time"

type ConceptoViaje struct {
	Codigo    string    `gorm:"primaryKey;size:50;not null" json:"codigo"`
	Nombre    string    `gorm:"size:100;not null;unique" json:"nombre"`
	CreatedAt time.Time `gorm:"type:timestamp" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp" json:"updated_at"`

	TiposSolicitud []TipoSolicitud `gorm:"foreignKey:ConceptoViajeCodigo;references:Codigo"`
}

func (ConceptoViaje) TableName() string { return "concepto_viajes" }
