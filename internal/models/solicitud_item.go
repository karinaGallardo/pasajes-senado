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

	Fecha        *time.Time           `gorm:"type:timestamp"`
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

func (t SolicitudItem) IsIda() bool {
	return t.Tipo == TipoSolicitudItemIda
}

func (t SolicitudItem) IsVuelta() bool {
	return t.Tipo == TipoSolicitudItemVuelta
}

func (t SolicitudItem) GetIcon() string {
	if t.Tipo == TipoSolicitudItemIda {
		return "ph-airplane-takeoff"
	}
	return "ph-airplane-landing"
}

func (t SolicitudItem) GetColorClass() string {
	if t.Tipo == TipoSolicitudItemIda {
		return "text-primary-600"
	}
	return "text-secondary-600"
}

func (t SolicitudItem) GetStatusBadgeClass() string {
	switch t.GetEstado() {
	case "SOLICITADO":
		return "bg-primary-100 text-primary-700 font-bold"
	case "PENDIENTE":
		return "bg-neutral-100 text-neutral-500 font-medium"
	case "APROBADO":
		return "bg-success-100 text-success-700 font-bold"
	case "EMITIDO":
		return "bg-secondary-100 text-secondary-700 font-bold"
	case "USADO":
		return "bg-neutral-800 text-white font-bold"
	case "RECHAZADO":
		return "bg-danger-100 text-danger-700 font-bold"
	case "REPROGRAMADO":
		return "bg-violet-100 text-violet-700 font-bold"
	default:
		return "bg-neutral-100 text-neutral-600"
	}
}

func (t SolicitudItem) HasActivePasaje() bool {
	for _, p := range t.Pasajes {
		if p.EstadoPasajeCodigo != nil && *p.EstadoPasajeCodigo != "ANULADO" {
			return true
		}
	}
	return false
}

func (t SolicitudItem) GetPasajeActivo() *Pasaje {
	if len(t.Pasajes) == 0 {
		return nil
	}
	// Devuelve el primer pasaje que no esté anulado (ahora solo debería haber uno activo por diseño)
	for i := range t.Pasajes {
		p := &t.Pasajes[i]
		if p.EstadoPasajeCodigo == nil || *p.EstadoPasajeCodigo != "ANULADO" {
			return p
		}
	}
	return nil
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

func (t SolicitudItem) CanBeApproved(user *Usuario) bool {
	if user == nil {
		return false
	}
	st := t.GetEstado()
	return user.IsAdminOrResponsable() && (st == "SOLICITADO" || st == "PENDIENTE")
}

func (t SolicitudItem) CanBeRejected(user *Usuario) bool {
	return t.CanBeApproved(user)
}

func (t SolicitudItem) CanBeReverted(user *Usuario) bool {
	if user == nil {
		return false
	}
	st := t.GetEstado()
	return user.IsAdminOrResponsable() && (st == "APROBADO" || st == "EMITIDO") && !t.HasActivePasaje()
}

func (t SolicitudItem) CanAssignPasaje(user *Usuario) bool {
	if user == nil {
		return false
	}
	st := t.GetEstado()
	// Un encargado/admin puede asignar pasaje si el tramo está APROBADO y no hay uno activo.
	return user.IsAdminOrResponsable() && st == "APROBADO" && !t.HasActivePasaje()
}

func (t SolicitudItem) GetOrigenDisplay() string {
	if t.Origen != nil {
		return t.Origen.Ciudad + " (" + t.Origen.IATA + ")"
	}
	return t.OrigenIATA
}

func (t SolicitudItem) GetDestinoDisplay() string {
	if t.Destino != nil {
		return t.Destino.Ciudad + " (" + t.Destino.IATA + ")"
	}
	return t.DestinoIATA
}
