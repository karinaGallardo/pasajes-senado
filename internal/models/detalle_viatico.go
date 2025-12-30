package models

import "time"

type DetalleViatico struct {
	BaseModel
	ViaticoID string `gorm:"size:36;not null;index"`

	FechaDesde time.Time `gorm:"not null"`
	FechaHasta time.Time `gorm:"not null"`

	Dias float64 `gorm:"type:decimal(5,2);not null"`

	Lugar string `gorm:"size:200"`

	MontoDia   float64 `gorm:"type:decimal(10,2);not null"`
	Porcentaje int     `gorm:"not null"`
	SubTotal   float64 `gorm:"type:decimal(10,2);not null"`

	CategoriaID *string `gorm:"size:36"`
}

func (DetalleViatico) TableName() string {
	return "detalle_viaticos"
}
