package models

import "time"

type DocumentoDescargo struct {
	BaseModel
	DescargoID string    `gorm:"size:36;not null;index"`
	Tipo       string    `gorm:"size:50;default:'FACTURA'"`
	Numero     string    `gorm:"size:100;not null"`
	Detalle    string    `gorm:"size:255"`
	Fecha      time.Time `gorm:"not null;type:timestamp"`
}

func (DocumentoDescargo) TableName() string {
	return "documentos_descargo"
}
