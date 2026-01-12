package models

import (
	"time"

	"gorm.io/gorm"
)

type Destino struct {
	IATA               string        `gorm:"primaryKey;size:5;not null" json:"iata"`
	Ciudad             string        `gorm:"size:100;not null;uniqueIndex:idx_destino_location" json:"ciudad"`
	Aeropuerto         string        `gorm:"size:255" json:"aeropuerto"`
	AmbitoCodigo       string        `gorm:"size:20;not null;index"`
	Ambito             *AmbitoViaje  `gorm:"foreignKey:AmbitoCodigo;references:Codigo"`
	DepartamentoCodigo *string       `gorm:"size:5;index;uniqueIndex:idx_destino_location"`
	Departamento       *Departamento `gorm:"foreignKey:DepartamentoCodigo;references:Codigo"`
	Pais               *string       `gorm:"size:100;uniqueIndex:idx_destino_location" json:"pais"`
	Estado             bool          `gorm:"default:true"`

	CreatedAt time.Time      `gorm:"index;type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`

	CreatedBy *string `gorm:"size:36;default:null"`
	UpdatedBy *string `gorm:"size:36;default:null"`
	DeletedBy *string `gorm:"size:36;default:null"`
}

func (Destino) TableName() string {
	return "destinos"
}

func (d Destino) GetNombreDisplay() string {
	if d.Aeropuerto != "" {
		return d.Ciudad + " - " + d.Aeropuerto + " (" + d.IATA + ")"
	}
	return d.Ciudad + " (" + d.IATA + ")"
}
