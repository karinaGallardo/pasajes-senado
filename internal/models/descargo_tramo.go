package models

import "time"

type TipoDescargoTramo string

const (
	TipoTramoIdaOriginal        TipoDescargoTramo = "IDA_ORIGINAL"
	TipoTramoIdaReprogramada    TipoDescargoTramo = "IDA_REPRO"
	TipoTramoVueltaOriginal     TipoDescargoTramo = "VUELTA_ORIGINAL"
	TipoTramoVueltaReprogramada TipoDescargoTramo = "VUELTA_REPRO"
)

type DescargoTramo struct {
	BaseModel
	DescargoID        string            `gorm:"size:36;not null;index"`
	Tipo              TipoDescargoTramo `gorm:"size:20;not null"`
	RutaID            *string           `gorm:"size:36;index"`
	RutaPasaje        *Ruta             `gorm:"foreignKey:RutaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	PasajeID          *string           `gorm:"size:36;index"`
	Pasaje            *Pasaje           `gorm:"foreignKey:PasajeID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	SolicitudItemID   *string           `gorm:"size:36;index"`
	SolicitudItem     *SolicitudItem    `gorm:"foreignKey:SolicitudItemID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`
	Fecha             *time.Time        `gorm:"type:timestamp"`
	Billete           string            `gorm:"size:100"`
	NumeroPaseAbordo  string            `gorm:"size:100"`
	ArchivoPaseAbordo string            `gorm:"size:255"`
	EsDevolucion      bool              `gorm:"default:false"`
	EsModificacion    bool              `gorm:"default:false"`
	MontoDevolucion   float64           `gorm:"type:decimal(10,2);default:0"`
	Moneda            string            `gorm:"size:10;default:'Bs.'"`
	Orden             int               `gorm:"default:0"`
}

func (d DescargoTramo) GetRutaDisplay() string {
	if d.RutaPasaje != nil {
		tramos := d.RutaPasaje.GetTramos()
		// If we have enough tramos, return the specific one for this order
		if len(tramos) > 0 && d.Orden < len(tramos) {
			return tramos[d.Orden]
		}
		return d.RutaPasaje.GetRutaDisplay()
	}
	return "Ruta no especificada"
}

func (DescargoTramo) TableName() string {
	return "descargo_tramos"
}

func (d DescargoTramo) GetTipoDisplay() string {
	switch d.Tipo {
	case TipoTramoIdaOriginal:
		return "IDA Original"
	case TipoTramoVueltaOriginal:
		return "Vuelta Original"
	case TipoTramoIdaReprogramada:
		return "IDA Repro"
	case TipoTramoVueltaReprogramada:
		return "Vuelta Repro"
	default:
		return string(d.Tipo)
	}
}

func (d DescargoTramo) GetTipoColorClass() string {
	switch d.Tipo {
	case TipoTramoIdaOriginal:
		return "bg-blue-50 text-blue-700 border-blue-100"
	case TipoTramoVueltaOriginal:
		return "bg-indigo-50 text-indigo-700 border-indigo-100"
	default:
		return "bg-amber-50 text-amber-700 border-amber-100"
	}
}
func (d DescargoTramo) GetPasajeCorrespondiente(s *Solicitud) *Pasaje {
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
	// 2. Fallback por Billete
	for _, item := range s.Items {
		for i := range item.Pasajes {
			p := &item.Pasajes[i]
			if p.NumeroBillete == d.Billete && p.NumeroBillete != "" {
				return p
			}
		}
	}
	return nil
}

func (d DescargoTramo) HasChanges(other DescargoTramo) bool {
	// Compare Main Fields
	if d.Tipo != other.Tipo ||
		d.Billete != other.Billete ||
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
