package models

import "time"

type TipoSolicitudItem string

const (
	TipoSolicitudItemIda    TipoSolicitudItem = "IDA"
	TipoSolicitudItemVuelta TipoSolicitudItem = "VUELTA"
)

type SolicitudItem struct {
	BaseModel
	SolicitudID string     `gorm:"size:36;not null;index"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	Tipo TipoSolicitudItem `gorm:"size:20;not null"` // IDA, VUELTA

	OrigenIATA string   `gorm:"size:5;not null"`
	Origen     *Destino `gorm:"foreignKey:OrigenIATA;references:IATA"`

	DestinoIATA string   `gorm:"size:5;not null"`
	Destino     *Destino `gorm:"foreignKey:DestinoIATA;references:IATA"`

	Fecha *time.Time `gorm:"type:timestamp"`
	EstadoCodigo *string              `gorm:"size:20;index;default:'SOLICITADO'"`
	Estado       *EstadoSolicitudItem `gorm:"foreignKey:EstadoCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	// Relation to Pasajes (History of tickets for this leg)
	Pasajes []Pasaje `gorm:"foreignKey:SolicitudItemID"`
}

func (SolicitudItem) TableName() string {
	return "solicitud_items"
}

func (t SolicitudItem) GetEstado() string {
	if t.EstadoCodigo == nil {
		return "SOLICITADO"
	}
	return *t.EstadoCodigo
}

func (t SolicitudItem) CanEdit() bool {
	st := t.GetEstado()
	return st == "PENDIENTE" || st == "SOLICITADO" || st == "RECHAZADO"
}

func (t SolicitudItem) HasActivePasaje() bool {
	for _, p := range t.Pasajes {
		if p.EstadoPasajeCodigo != nil && *p.EstadoPasajeCodigo != "ANULADO" {
			return true
		}
	}
	return false
}

func (t SolicitudItem) GetPasajeOriginal() *Pasaje {
	if len(t.Pasajes) == 0 {
		return nil
	}
	// The original is usually the first one or the one without PasajeAnteriorID
	for i := range t.Pasajes {
		p := &t.Pasajes[i]
		if p.PasajeAnteriorID == nil && (p.EstadoPasajeCodigo == nil || *p.EstadoPasajeCodigo != "ANULADO") {
			return p
		}
	}
	if len(t.Pasajes) > 0 {
		p := &t.Pasajes[0]
		if p.EstadoPasajeCodigo == nil || *p.EstadoPasajeCodigo != "ANULADO" {
			return p
		}
	}
	return nil
}

func (t SolicitudItem) GetPasajeReprogramado() *Pasaje {
	if len(t.Pasajes) < 2 {
		return nil
	}
	// The reprogrammed is the latest active one that is not the original
	var latest *Pasaje
	original := t.GetPasajeOriginal()
	for i := range t.Pasajes {
		p := &t.Pasajes[i]
		if original != nil && p.ID == original.ID {
			continue
		}
		if p.EstadoPasajeCodigo != nil && *p.EstadoPasajeCodigo != "ANULADO" {
			latest = p
		}
	}
	return latest
}

// GetChanges compares current item with old state and returns dirty fields map for GORM Updates
func (t *SolicitudItem) GetChanges(old SolicitudItem) map[string]any {
	changes := make(map[string]any)

	if t.OrigenIATA != old.OrigenIATA {
		changes["origen_iata"] = t.OrigenIATA
	}
	if t.DestinoIATA != old.DestinoIATA {
		changes["destino_iata"] = t.DestinoIATA
	}

	// Comparar estados
	if (t.EstadoCodigo == nil) != (old.EstadoCodigo == nil) ||
		(t.EstadoCodigo != nil && old.EstadoCodigo != nil && *t.EstadoCodigo != *old.EstadoCodigo) {
		changes["estado_codigo"] = t.EstadoCodigo
	}

	// Comparar fechas usando Segundos Unix para evitar líos de precisión
	if (t.Fecha == nil) != (old.Fecha == nil) ||
		(t.Fecha != nil && old.Fecha != nil && t.Fecha.Unix() != old.Fecha.Unix()) {
		changes["fecha"] = t.Fecha
	}

	return changes
}
