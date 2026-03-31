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
	RutaID            *string               `gorm:"size:36;index"`
	RutaPasaje        *Ruta                 `gorm:"foreignKey:RutaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	PasajeID          *string               `gorm:"size:36;index"`
	Pasaje            *Pasaje               `gorm:"foreignKey:PasajeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	SolicitudItemID   *string               `gorm:"size:36;index"`
	SolicitudItem     *SolicitudItem        `gorm:"foreignKey:SolicitudItemID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	Fecha             *time.Time            `gorm:"type:timestamp"`
	Boleto            string                `gorm:"size:100"`
	NumeroPaseAbordo  string                `gorm:"size:100"`
	ArchivoPaseAbordo string                `gorm:"size:255"`
	EsDevolucion      bool                  `gorm:"default:false"`
	EsModificacion    bool                  `gorm:"default:false"`
	MontoDevolucion   float64               `gorm:"type:decimal(10,2);default:0"`
	Moneda            string                `gorm:"size:10;default:'Bs.'"`
	Orden             int                   `gorm:"default:0"`
}

func (d DetalleItinerarioDescargo) GetRutaDisplay() string {
	if d.RutaPasaje != nil {
		segments := d.RutaPasaje.GetSegments()
		// If it's a legacy record or we don't have enough segments, fall back to total display
		if len(segments) > 1 && d.Orden < len(segments) {
			return segments[d.Orden]
		}
		return d.RutaPasaje.GetTramoDisplay()
	}
	return "Ruta no especificada"
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
	// 1. Prioridad por ID
	if d.PasajeID != nil {
		for _, item := range s.Items {
			for i := range item.Pasajes {
				if item.Pasajes[i].ID == *d.PasajeID {
					return &item.Pasajes[i]
				}
			}
		}
	}
	// 2. Fallback por Boleto
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

func (d DetalleItinerarioDescargo) HasChanges(other DetalleItinerarioDescargo) bool {
	// Compare Main Fields
	if d.Tipo != other.Tipo ||
		d.Boleto != other.Boleto ||
		d.NumeroPaseAbordo != other.NumeroPaseAbordo ||
		d.ArchivoPaseAbordo != other.ArchivoPaseAbordo ||
		d.EsDevolucion != other.EsDevolucion ||
		d.EsModificacion != other.EsModificacion ||
		d.MontoDevolucion != other.MontoDevolucion ||
		d.Moneda != other.Moneda ||
		d.Orden != other.Orden {
		return true
	}

	// Compare Pointers (Dates)
	if (d.Fecha == nil) != (other.Fecha == nil) {
		return true
	}

	if d.Fecha != nil && other.Fecha != nil && !d.Fecha.Equal(*other.Fecha) {
		return true
	}

	// Compare Pointers (IDs)
	cmpPtr := func(p1, p2 *string) bool {
		if (p1 == nil) != (p2 == nil) {
			return true // One is nil, the other isn't
		}
		if p1 != nil && p2 != nil && *p1 != *p2 {
			return true // Both are present but different
		}
		return false
	}

	if cmpPtr(d.RutaID, other.RutaID) ||
		cmpPtr(d.PasajeID, other.PasajeID) ||
		cmpPtr(d.SolicitudItemID, other.SolicitudItemID) {
		return true
	}

	return false
}
