package models

import "time"

type Pasaje struct {
	BaseModel
	SolicitudID string `gorm:"not null;size:36"`

	AerolineaID *string    `gorm:"size:36"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID"`

	AgenciaID *string  `gorm:"size:36"`
	Agencia   *Agencia `gorm:"foreignKey:AgenciaID"`

	NumeroVuelo string `gorm:"size:50"`
	Ruta        string `gorm:"size:255"`

	FechaVuelo time.Time `gorm:"type:timestamp"`

	CodigoReserva string  `gorm:"size:50"`
	NumeroBoleto  string  `gorm:"size:100;index"`
	Costo         float64 `gorm:"type:decimal(10,2)"`

	EstadoPasajeCodigo *string       `gorm:"size:50;default:'EMITIDO'"`
	EstadoPasaje       *EstadoPasaje `gorm:"foreignKey:EstadoPasajeCodigo"`

	Archivo string `gorm:"size:255;default:''"`

	PasajeAnteriorID *string `gorm:"size:36"`
	PasajeAnterior   *Pasaje `gorm:"foreignKey:PasajeAnteriorID"`
	Glosa            string  `gorm:"type:text"`

	NumeroFactura  string  `gorm:"size:50;index"`
	CostoPenalidad float64 `gorm:"type:decimal(10,2);default:0"`
}

func (Pasaje) TableName() string {
	return "pasajes"
}
