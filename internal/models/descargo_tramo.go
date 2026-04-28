package models

import (
	"strings"
	"time"
)

type TipoDescargoTramo string

const (
	TipoTramoIdaOriginal        TipoDescargoTramo = "IDA_ORIGINAL"
	TipoTramoIdaReprogramada    TipoDescargoTramo = "IDA_REPRO"
	TipoTramoIdaReutilizada     TipoDescargoTramo = "IDA_REUT"
	TipoTramoVueltaOriginal     TipoDescargoTramo = "VUELTA_ORIGINAL"
	TipoTramoVueltaReprogramada TipoDescargoTramo = "VUELTA_REPRO"
	TipoTramoVueltaReutilizada  TipoDescargoTramo = "VUELTA_REUT"
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
	NumeroVuelo       string            `gorm:"size:20"`
	NumeroPaseAbordo  string            `gorm:"size:100"`
	ArchivoPaseAbordo string            `gorm:"size:255"`
	EsOpenTicket      bool              `gorm:"default:false"`
	EsModificacion    bool              `gorm:"default:false"`
	EsReutilizado     bool              `gorm:"default:false"`

	OrigenIATA  *string  `gorm:"size:5;index" json:"origen_iata"`
	Origen      *Destino `gorm:"foreignKey:OrigenIATA;references:IATA;<-:false" json:"origen"`
	DestinoIATA *string  `gorm:"size:5;index" json:"destino_iata"`
	Destino     *Destino `gorm:"foreignKey:DestinoIATA;references:IATA;<-:false" json:"destino"`

	// TramoNombre helps to store the specific leg of a journey as a string (fallback)
	TramoNombre string `gorm:"size:255" json:"tramo_nombre"`

	// Sequence field to maintain deterministic order
	Seq int `gorm:"index;default:1" json:"seq"`
}

func (d DescargoTramo) GetRutaDisplay() string {
	if d.Origen != nil && d.Destino != nil {
		return d.Origen.GetNombreCorto() + " - " + d.Destino.GetNombreCorto()
	}
	if d.OrigenIATA != nil && d.DestinoIATA != nil {
		return *d.OrigenIATA + " - " + *d.DestinoIATA
	}
	if d.TramoNombre != "" {
		return d.TramoNombre
	}
	if d.RutaPasaje != nil {
		return d.RutaPasaje.GetRutaDisplay()
	}
	return "Ruta no especificada"
}

func (d DescargoTramo) GetOrigenIATA() string {
	if d.OrigenIATA != nil {
		return *d.OrigenIATA
	}
	return ""
}

func (d DescargoTramo) GetDestinoIATA() string {
	if d.DestinoIATA != nil {
		return *d.DestinoIATA
	}
	return ""
}

func (DescargoTramo) TableName() string {
	return "descargo_tramos"
}

// IsOriginal returns true if this tramo is an original segment (not a reprogrammed or returned one).
func (d DescargoTramo) IsOriginal() bool {
	return strings.HasSuffix(string(d.Tipo), "_ORIGINAL")
}

// IsReprogramacion returns true if this tramo is a reprogrammed segment.
func (d DescargoTramo) IsReprogramacion() bool {
	upper := strings.ToUpper(string(d.Tipo))
	return strings.HasSuffix(upper, "_REPRO") || strings.HasSuffix(upper, "_REPROG")
}

// IsReutilizacion returns true if this tramo is a reused segment (REUT).
func (d DescargoTramo) IsReutilizacion() bool {
	return strings.HasSuffix(strings.ToUpper(string(d.Tipo)), "_REUT")
}

// GetRutaOrigen extracts the origin from the routing label.
func (d DescargoTramo) GetRutaOrigen() string {
	display := d.GetRutaDisplay()
	parts := strings.Split(display, " - ")
	if len(parts) >= 2 {
		return parts[0]
	}
	return display
}

// GetRutaDestino extracts the destination from the routing label.
func (d DescargoTramo) GetRutaDestino() string {
	display := d.GetRutaDisplay()
	parts := strings.Split(display, " - ")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// GetFechaStr returns the date formatted as string.
func (d DescargoTramo) GetFechaStr() string {
	if d.Fecha != nil {
		return d.Fecha.Format("2006-01-02 15:04")
	}
	return ""
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
		d.TramoNombre != other.TramoNombre ||
		d.NumeroVuelo != other.NumeroVuelo ||
		d.NumeroPaseAbordo != other.NumeroPaseAbordo ||
		d.ArchivoPaseAbordo != other.ArchivoPaseAbordo ||
		d.EsOpenTicket != other.EsOpenTicket ||
		d.EsModificacion != other.EsModificacion ||
		d.EsReutilizado != other.EsReutilizado ||
		d.Seq != other.Seq {
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
		cmpPtr(d.SolicitudItemID, other.SolicitudItemID) ||
		cmpPtr(d.OrigenIATA, other.OrigenIATA) ||
		cmpPtr(d.DestinoIATA, other.DestinoIATA) {
		return true
	}

	return false
}
