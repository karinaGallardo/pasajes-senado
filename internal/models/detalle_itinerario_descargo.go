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
	EsDevolucion      bool                  `gorm:"default:false"`
	EsModificacion    bool                  `gorm:"default:false"`
	Orden             int                   `gorm:"default:0"`
}

func (DetalleItinerarioDescargo) TableName() string {
	return "detalle_itinerario_descargos"
}

func (d DetalleItinerarioDescargo) GetTipoDisplay() string {
	switch d.Tipo {
	case TipoDetalleIdaOriginal:
		return "IDA Original"
	case TipoDetalleVueltaOriginal:
		return "Vuelta Original"
	case TipoDetalleIdaReprogramada:
		return "IDA Repro"
	case TipoDetalleVueltaReprogramada:
		return "Vuelta Repro"
	default:
		return string(d.Tipo)
	}
}

func (d DetalleItinerarioDescargo) GetTipoColorClass() string {
	switch d.Tipo {
	case TipoDetalleIdaOriginal:
		return "bg-blue-50 text-blue-700 border-blue-100"
	case TipoDetalleVueltaOriginal:
		return "bg-indigo-50 text-indigo-700 border-indigo-100"
	default:
		return "bg-amber-50 text-amber-700 border-amber-100"
	}
}
func (d DetalleItinerarioDescargo) GetPasajeCorrespondiente(s *Solicitud) *Pasaje {
	if s == nil {
		return nil
	}
	for _, item := range s.Items {
		for i := range item.Pasajes {
			p := &item.Pasajes[i]
			if p.NumeroBoleto == d.Boleto && p.NumeroBoleto != "" {
				return p
			}
		}
	}
	return nil
}
