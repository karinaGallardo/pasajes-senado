package models

import (
	"fmt"
	"time"
)

type TipoSolicitudItem string

const (
	TipoSolicitudItemIda    TipoSolicitudItem = "IDA"
	TipoSolicitudItemVuelta TipoSolicitudItem = "VUELTA"
)

type SolicitudItemPermissions struct {
	CanEdit         bool
	CanApprove      bool
	CanReject       bool
	CanRevert       bool
	CanAssignPasaje bool
}

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

	AerolineaID *string    `gorm:"size:36;index;default:null"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`

	// Relation to Pasajes (History of tickets for this leg)
	Pasajes []Pasaje `gorm:"foreignKey:SolicitudItemID"`

	// Contexto de runtime (no persistido)
	authUser    *Usuario                  `gorm:"-"`
	Permissions *SolicitudItemPermissions `gorm:"-"`
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

func (t SolicitudItem) GetPermissions(u ...*Usuario) SolicitudItemPermissions {
	user := t.getAuthUser(u...)
	return SolicitudItemPermissions{
		CanEdit:         t.CanEdit(),
		CanApprove:      t.CanBeApproved(user),
		CanReject:       t.CanBeRejected(user),
		CanRevert:       t.CanBeReverted(user),
		CanAssignPasaje: t.CanAssignPasaje(user),
	}
}

func (t *SolicitudItem) HydratePermissions(u ...*Usuario) {
	if len(u) > 0 {
		t.authUser = u[0]
	}
	p := t.GetPermissions()
	t.Permissions = &p
}

func (t SolicitudItem) getAuthUser(u ...*Usuario) *Usuario {
	if len(u) > 0 {
		return u[0]
	}
	return t.authUser
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

func (t SolicitudItem) IsPendiente() bool {
	return t.GetEstado() == "PENDIENTE"
}

func (t SolicitudItem) IsSolicitado() bool {
	return t.GetEstado() == "SOLICITADO"
}

func (t SolicitudItem) IsAprobado() bool {
	return t.GetEstado() == "APROBADO"
}

func (t SolicitudItem) IsRechazado() bool {
	return t.GetEstado() == "RECHAZADO"
}

func (t SolicitudItem) IsEmitido() bool {
	return t.GetEstado() == "EMITIDO"
}

func (t SolicitudItem) IsFinalizado() bool {
	return t.GetEstado() == "FINALIZADO"
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
	if t.Estado != nil && t.Estado.Color != "" {
		color := t.Estado.Color
		// Casos especiales (neutral black, USADO dark, etc)
		if color == "neutral-800" || color == "black" {
			return "bg-neutral-800 text-white font-bold"
		}
		return fmt.Sprintf("bg-%s-100 text-%s-700 font-bold", color, color)
	}

	// Fallback por defecto
	return "bg-neutral-100 text-neutral-600"
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
	if t.AerolineaID != old.AerolineaID {
		changes["aerolinea_id"] = t.AerolineaID
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
	// Un encargado/admin puede asignar pasaje siempre que el tramo esté APROBADO o ya tenga emisiones previas (EMITIDO).
	// Esto permite el registro de múltiples billetes para tramos compuestos (multi-aerolínea).
	return user.IsAdminOrResponsable() && (st == "APROBADO" || st == "EMITIDO")
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

// GetCostoTotal suma el costo de todos los pasajes asociados a este tramo que no estén anulados.
func (t SolicitudItem) GetCostoTotal() float64 {
	total := 0.0
	for _, p := range t.Pasajes {
		estado := p.GetEstadoCodigo()
		// No sumamos pasajes anulados o que no tengan costo registrado
		if estado != "ANULADO" {
			total += p.Costo
		}
	}
	return total
}

// Actions

func (t *SolicitudItem) Approve() {
	st := "APROBADO"
	t.EstadoCodigo = &st
}

func (t *SolicitudItem) Reject() {
	st := "RECHAZADO"
	t.EstadoCodigo = &st
}

func (t *SolicitudItem) Finalize() {
	st := "FINALIZADO"
	t.EstadoCodigo = &st
	for i := range t.Pasajes {
		p := &t.Pasajes[i]
		if p.GetEstadoCodigo() == "EMITIDO" {
			stP := "USADO"
			p.EstadoPasajeCodigo = &stP
		}
	}
}

func (t *SolicitudItem) RevertApproval() {
	st := "SOLICITADO"
	t.EstadoCodigo = &st
}

func (t *SolicitudItem) RevertFinalize() {
	st := "EMITIDO"
	t.EstadoCodigo = &st
	for i := range t.Pasajes {
		p := &t.Pasajes[i]
		if p.GetEstadoCodigo() == "USADO" {
			stP := "EMITIDO"
			p.EstadoPasajeCodigo = &stP
		}
	}
}

func NewSolicitudItem(solicitudID string, tipoStr string, origen, destino string, fecha *time.Time, aerolineaID *string) *SolicitudItem {
	tipo := TipoSolicitudItemIda
	if tipoStr == "VUELTA" {
		tipo = TipoSolicitudItemVuelta
	}

	st := "SOLICITADO"
	if tipo == TipoSolicitudItemVuelta && fecha == nil {
		st = "PENDIENTE"
	}

	return &SolicitudItem{
		SolicitudID:  solicitudID,
		Tipo:         tipo,
		OrigenIATA:   origen,
		DestinoIATA:  destino,
		Fecha:        fecha,
		EstadoCodigo: &st,
		AerolineaID:  aerolineaID,
	}
}
