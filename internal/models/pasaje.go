package models

import (
	"fmt"
	"time"
)

// PasajePermissions define las acciones permitidas sobre un pasaje según el rol y estado.
type PasajePermissions struct {
	CanEdit            bool
	CanMarkUsado       bool
	CanRevertirEmision bool
	CanEmitir          bool
	CanValidateUso     bool
	ShowActionsMenu    bool
}

type Pasaje struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;index"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	SolicitudItemID *string        `gorm:"size:36;index"`
	SolicitudItem   *SolicitudItem `gorm:"foreignKey:SolicitudItemID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`

	AerolineaID *string    `gorm:"size:36"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	AgenciaID *string  `gorm:"size:36"`
	Agencia   *Agencia `gorm:"foreignKey:AgenciaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	NumeroVuelo string `gorm:"size:50"`

	RutaID     *string `gorm:"size:36;index"`
	RutaPasaje *Ruta   `gorm:"foreignKey:RutaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	FechaVuelo   time.Time  `gorm:"type:timestamp"`
	FechaEmision *time.Time `gorm:"type:date"`

	CodigoReserva    string  `gorm:"size:50"`
	NumeroBillete    string  `gorm:"size:100;index"`
	Costo            float64 `gorm:"type:decimal(10,2)"`
	CostoUtilizacion float64 `gorm:"type:decimal(10,2);default:0"`
	Diferencia       float64 `gorm:"type:decimal(10,2);default:0"`

	EstadoPasajeCodigo *string       `gorm:"size:50;default:'EMITIDO'"`
	EstadoPasaje       *EstadoPasaje `gorm:"foreignKey:EstadoPasajeCodigo;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	Archivo string `gorm:"size:255;default:''"`

	ArchivoPaseAbordo string  `gorm:"size:255;default:''"`
	Glosa             string  `gorm:"type:text"`
	NumeroFactura     string  `gorm:"size:50;index"`
	CostoPenalidad    float64 `gorm:"type:decimal(10,2);default:0"`

	// Servicio de Emisión (Fee)
	CostoServicioEmision  float64 `gorm:"type:decimal(10,2);default:0"`
	NroFacturaEmision     string  `gorm:"size:50;index"`
	ArchivoFacturaEmision string  `gorm:"size:255;default:''"`

	// Seq is an auto-incrementing field managed by DB to ensure atomic sequential ordering
	Seq int64 `gorm:"autoIncrement;not null;<-:false"`

	// Contexto de runtime (no persistido)
	authUser    *Usuario           `gorm:"-"`
	Permissions *PasajePermissions `gorm:"-"`
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

func (Pasaje) TableName() string {
	return "pasajes"
}

func (p Pasaje) GetEstado() string {
	if p.EstadoPasaje != nil {
		return p.EstadoPasaje.Codigo
	}
	if p.EstadoPasajeCodigo == nil {
		return "EMITIDO"
	}
	return *p.EstadoPasajeCodigo
}

func (p Pasaje) GetEstadoCodigo() string {
	if p.EstadoPasajeCodigo == nil {
		return ""
	}
	return *p.EstadoPasajeCodigo
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
	return user.IsAdminOrResponsable() && p.GetEstado() == "REGISTRADO"
}

func (p Pasaje) CanBeEmitted(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "REGISTRADO"
}

func (p Pasaje) CanBeReverted(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	if user == nil {
		return false
	}
	return user.IsAdminOrResponsable() && p.GetEstado() == "EMITIDO"
}

func (p Pasaje) CanMarkUsado(u ...*Usuario) bool {
	user := p.getAuthUser(u...)
	// Necesitamos la solicitud para el contexto de dueño/creador
	sol := p.Solicitud
	if user == nil || sol == nil {
		return false
	}

	// Solo pasajes emitidos pueden marcarse como usados
	if p.GetEstado() != "EMITIDO" {
		return false
	}
	// Admin/Responsable puede siempre
	if user.IsAdminOrResponsable() {
		return true
	}
	// El dueño de la solicitud
	if user.ID == sol.UsuarioID {
		return true
	}
	// El creador
	if sol.CreatedBy != nil && *sol.CreatedBy == user.ID {
		return true
	}
	// El asistente/encargado del usuario
	if sol.Usuario.EncargadoID != nil && *sol.Usuario.EncargadoID == user.ID {
		return true
	}
	return false
}

func (p Pasaje) IsDischargeable() bool {
	st := p.GetEstadoCodigo()
	return st == "EMITIDO" || st == "USADO"
}

func (p Pasaje) GetStatusBannerClass() string {
	switch p.GetEstado() {
	case "EMITIDO":
		return "bg-success-600"
	case "ANULADO":
		return "bg-neutral-600"
	case "USADO":
		return "bg-primary-600"
	default:
		return "bg-secondary-600"
	}
}

// GetPermissions calcula el conjunto de permisos para un usuario específico sobre este pasaje.
func (p Pasaje) GetPermissions(u ...*Usuario) PasajePermissions {
	perms := PasajePermissions{
		CanEdit:            p.CanBeEdited(u...),
		CanMarkUsado:       p.CanMarkUsado(u...),
		CanRevertirEmision: p.CanBeReverted(u...),
		CanEmitir:          p.CanBeEmitted(u...),
		CanValidateUso:     false, // Campo deprecado o para uso futuro
	}
	perms.ShowActionsMenu = perms.CanEdit || perms.CanMarkUsado || perms.CanRevertirEmision || perms.CanEmitir
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
		return fmt.Sprintf("bg-%s-100 text-%s-800", p.EstadoPasaje.Color, p.EstadoPasaje.Color)
	}
	switch p.GetEstado() {
	case "REGISTRADO":
		return "bg-secondary-100 text-secondary-800"
	case "EMITIDO":
		return "bg-success-100 text-success-800"
	case "USADO":
		return "bg-primary-100 text-primary-800"
	case "ANULADO":
		return "bg-neutral-100 text-neutral-800"
	default:
		return "bg-neutral-100 text-neutral-800"
	}
}
