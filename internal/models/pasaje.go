package models

import "time"

type Pasaje struct {
	ModeloBase
	SolicitudID uint `gorm:"not null"`

	Aerolinea   string `gorm:"size:100"`
	NumeroVuelo string `gorm:"size:50"`
	Ruta        string `gorm:"size:255"`
	FechaVuelo  time.Time

	CodigoReserva string  `gorm:"size:50"`
	NumeroBoleto  string  `gorm:"size:100;index"`
	Costo         float64 `gorm:"type:decimal(10,2)"`

	Estado string `gorm:"size:50;default:'EMITIDO'"`
}

func (Pasaje) TableName() string {
	return "pasajes"
}
