package models

import "time"

type Descargo struct {
	ModeloBase
	SolicitudID uint   `gorm:"not null;uniqueIndex"`
	UsuarioID   string `gorm:"size:24;not null"`

	FechaPresentacion  time.Time `gorm:"not null"`
	InformeActividades string    `gorm:"type:text"`

	MontoDevolucion float64 `gorm:"type:decimal(10,2);default:0"`
	Observaciones   string  `gorm:"type:text"`

	Estado string `gorm:"size:50;default:'EN_REVISION'"`
}

func (Descargo) TableName() string {
	return "descargos"
}
