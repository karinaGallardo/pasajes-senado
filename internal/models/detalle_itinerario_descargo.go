package models

import "time"

type TipoDetalleItinerario string

const (
	TipoDetalleIdaOriginal        TipoDetalleItinerario = "IDA_ORIGINAL"
	TipoDetalleIdaReprogramada    TipoDetalleItinerario = "IDA_REPRO"
	TipoDetalleVueltaOriginal     TipoDetalleItinerario = "VUELTA_ORIGINAL"
	TipoDetalleVueltaReprogramada TipoDetalleItinerario = "VUELTA_REPRO"
)

type DetalleItinerarioDescargo struct {
	BaseModel
	DescargoID        string                `gorm:"size:36;not null;index"`
	Tipo              TipoDetalleItinerario `gorm:"size:20;not null"`
	Ruta              string                `gorm:"size:255"`
	Fecha             *time.Time            `gorm:"type:timestamp"`
	Boleto            string                `gorm:"size:100"`
	NumeroPaseAbordo  string                `gorm:"size:100"`
	ArchivoPaseAbordo string                `gorm:"size:255"`
	Orden             int                   `gorm:"default:0"`
}

func (DetalleItinerarioDescargo) TableName() string {
	return "detalle_itinerario_descargos"
}
