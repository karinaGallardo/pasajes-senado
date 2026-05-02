package models

import (
	"fmt"
	"strings"
	"time"
)

const (
	EstadoPasajeRegistrado = "REGISTRADO"
	EstadoPasajeEmitido    = "EMITIDO"
	EstadoPasajeFinalizado = "FINALIZADO"
)

// PasajePermissions define las acciones permitidas sobre un pasaje según el rol y estado.
type PasajePermissions struct {
	CanEdit            bool
	CanMarkUsado       bool
	CanRevertirEmision bool
	CanEmitir          bool
	CanValidateUso     bool
	CanDelete          bool
	ShowActionsMenu    bool
}

type Pasaje struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;index"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	SolicitudItemID *string        `gorm:"size:36;index"`
	SolicitudItem   *SolicitudItem `gorm:"foreignKey:SolicitudItemID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	AerolineaID *string    `gorm:"size:36"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	AgenciaID *string  `gorm:"size:36"`
	Agencia   *Agencia `gorm:"foreignKey:AgenciaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	NumeroVuelo string `gorm:"size:50"`

	RutaID     *string `gorm:"size:36;index"`
	RutaPasaje *Ruta   `gorm:"foreignKey:RutaID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	FechaVuelo   time.Time  `gorm:"type:timestamp"`
	FechaEmision *time.Time `gorm:"type:date"`

	NumeroBillete  string  `gorm:"size:100;index"`
	Costo          float64 `gorm:"type:decimal(10,2)"`
	CostoUtilizado float64 `gorm:"type:decimal(10,2);default:0" json:"costo_utilizado"`
	MontoReembolso float64 `gorm:"type:decimal(10,2);default:0" json:"monto_reembolso"`

	EstadoPasajeCodigo string        `gorm:"size:50;not null;default:'REGISTRADO'"`
	EstadoPasaje       *EstadoPasaje `gorm:"foreignKey:EstadoPasajeCodigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	Archivo string `gorm:"size:255;default:''"`

	ArchivoPaseAbordo string  `gorm:"size:255;default:''"`
	Glosa             string  `gorm:"type:text"`
	NumeroFactura     string  `gorm:"size:50;index"`
	CostoPenalidad    float64 `gorm:"type:decimal(10,2);default:0"`

	// Datos de Servicio de Emisión Agencia (Opcional)
	ServicioRazonSocial   string     `gorm:"type:varchar(255)"`
	ServicioFacturaNumero string     `gorm:"type:varchar(100);index"`
	ServicioFacturaFecha  *time.Time `gorm:"type:timestamp"`
	ServicioMonto         float64    `gorm:"type:decimal(10,2);default:0"`
	ServicioArchivo       string     `gorm:"type:varchar(255);default:''"`

	// Devolución por Diferencia de Tarifa (Per Pasaje)
	NroBoletaDeposito  string     `gorm:"size:100;index"`
	ArchivoComprobante string     `gorm:"size:255;default:''"`
	FechaDeposito      *time.Time `gorm:"type:timestamp"`

	// Cargos Asociados al Pasaje (Facturas de Emisión, Cambios, etc.)
	Cargos []PasajeCargo `gorm:"foreignKey:PasajeID"`

	// Relación inversa para liquidación
	DescargoTramos []DescargoTramo `gorm:"foreignKey:PasajeID;<-:false"`

	// Si este pasaje se emitió usando un Open Ticket previo
	OpenTicketID *string     `gorm:"size:36;index;default:null"`
	OpenTicket   *OpenTicket `gorm:"foreignKey:OpenTicketID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`

	// Contexto de runtime (no persistido)
	authUser    *Usuario           `gorm:"-"`
	Permissions *PasajePermissions `gorm:"-"`
}

func (Pasaje) TableName() string {
	return "pasajes"
}

func (p Pasaje) GetRutaDisplay() string {
	if p.RutaPasaje != nil {
		return p.RutaPasaje.GetRutaDisplay()
	}
	return "Ruta no especificada"
}

func (p Pasaje) GetTramosLegs() []TramoLeg {
	if p.RutaPasaje != nil {
		return p.RutaPasaje.GetTramosItems()
	}
	return []TramoLeg{}
}

func (p Pasaje) GetEstado() string {
	if p.EstadoPasaje != nil {
		return p.EstadoPasaje.Codigo
	}
	if p.EstadoPasajeCodigo == "" {
		return EstadoPasajeRegistrado
	}
	return p.EstadoPasajeCodigo
}

func (p Pasaje) GetEstadoCodigo() string {
	return p.EstadoPasajeCodigo
}

func (p Pasaje) getAuthUser(u ...*Usuario) *Usuario {
	if len(u) > 0 {
		return u[0]
	}
	return p.authUser
}

func (p Pasaje) CanBeEdited(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == EstadoPasajeRegistrado
}

func (p Pasaje) CanBeEmitted(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == EstadoPasajeRegistrado
}

func (p Pasaje) CanBeDeleted(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	// Solo se puede eliminar si está registrado (borrador)
	return user.IsAdminOrResponsable() && p.GetEstado() == EstadoPasajeRegistrado
}

func (p Pasaje) CanBeReverted(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == EstadoPasajeEmitido
}

func (p Pasaje) CanMarkFinalizado(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	// Solo Responsable puede marcar como FINALIZADO tras el descargo
	return user.IsResponsable() && p.GetEstado() == EstadoPasajeEmitido
}

func (p Pasaje) IsFinalizado() bool {
	st := p.GetEstado()
	return st == EstadoPasajeFinalizado
}

func (p Pasaje) IsDischargeable() bool {
	st := p.GetEstadoCodigo()
	return st == EstadoPasajeEmitido || st == EstadoPasajeFinalizado
}

// HasOpenTicket retorna true si este pasaje tiene al menos un tramo en el descargo marcado como Open Ticket
func (p Pasaje) HasOpenTicket() bool {
	for _, t := range p.DescargoTramos {
		if t.EsOpenTicket {
			return true
		}
	}
	return false
}

func (p Pasaje) GetMontoCargos() float64 {
	total := 0.0
	for _, c := range p.Cargos {
		total += c.Monto
	}
	return total
}

func (p Pasaje) GetStatusBannerClass() string {
	switch p.GetEstado() {
	case EstadoPasajeEmitido:
		return "bg-success-600"
	case EstadoPasajeFinalizado:
		return "bg-neutral-800"
	default:
		return "bg-secondary-600"
	}
}

// GetPermissions calcula el conjunto de permisos para un usuario específico sobre este pasaje.
func (p Pasaje) GetPermissions(u ...*Usuario) PasajePermissions {
	perms := PasajePermissions{
		CanEdit:            p.CanBeEdited(u...),
		CanMarkUsado:       p.CanMarkFinalizado(u...),
		CanRevertirEmision: p.CanBeReverted(u...),
		CanEmitir:          p.CanBeEmitted(u...),
		CanValidateUso:     false,
		CanDelete:          p.CanBeDeleted(u...),
	}
	perms.ShowActionsMenu = perms.CanEdit || perms.CanMarkUsado || perms.CanRevertirEmision || perms.CanEmitir || perms.CanDelete
	return perms
}

func (p *Pasaje) HydratePermissions(u ...*Usuario) {
	if len(u) > 0 {
		p.authUser = u[0]
	}
	perms := p.GetPermissions()
	p.Permissions = &perms
}

// GetStatusBadgeClass retorna las clases de CSS para los badges según el estado del pasaje.
func (p Pasaje) GetStatusBadgeClass() string {
	if p.EstadoPasaje != nil {
		color := p.EstadoPasaje.Color
		if strings.HasPrefix(color, "#") {
			return fmt.Sprintf("bg-[%s] text-white font-bold px-2 py-0.5 rounded shadow-sm", color)
		}
		return fmt.Sprintf("bg-%s-100 text-%s-800", color, color)
	}
	switch p.GetEstado() {
	case EstadoPasajeRegistrado:
		return "bg-secondary-100 text-secondary-800"
	case EstadoPasajeEmitido:
		return "bg-success-100 text-success-800"
	case EstadoPasajeFinalizado:
		return "bg-neutral-800 text-white font-bold"
	default:
		return "bg-neutral-100 text-neutral-800"
	}
}
