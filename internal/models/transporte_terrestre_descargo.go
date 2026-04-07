package models

import "time"

type TransporteTerrestreDescargo struct {
	BaseModel
	DescargoOficialID string `gorm:"size:36;not null;index"`

	Fecha      time.Time `gorm:"type:date"`
	NroFactura string    `gorm:"size:100"`
	Importe    float64   `gorm:"type:decimal(10,2)"`
	Tipo       string    `gorm:"size:20;default:'IDA'"` // IDA, VUELTA
	Archivo    string    `gorm:"size:255"`              // PDF comprobante

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`
}

func (TransporteTerrestreDescargo) TableName() string {
	return "transporte_terrestre_descargo"
}
