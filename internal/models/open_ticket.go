package models

import (
	"time"

	"gorm.io/gorm"
)

type EstadoOpenTicket string

const (
	EstadoOpenTicketPendiente  EstadoOpenTicket = "PENDIENTE"  // Generado en descargo, esperando aprobación
	EstadoOpenTicketDisponible EstadoOpenTicket = "DISPONIBLE" // Listo para ser usado
	EstadoOpenTicketReservado  EstadoOpenTicket = "RESERVADO"  // Seleccionado en una nueva solicitud FV-01
	EstadoOpenTicketFinalizado EstadoOpenTicket = "FINALIZADO" // Ya volado o canjeado definitivamente
	EstadoOpenTicketCancelado  EstadoOpenTicket = "CANCELADO"  // Anulado o corregido
)

// OpenTicket representa un billete aéreo que quedó sin usar
// y está disponible para ser reutilizado en un futuro viaje.
type OpenTicket struct {
	ID string `gorm:"primaryKey;type:uuid;default:uuidv7()" json:"id"`

	UsuarioID string   `gorm:"type:uuid;not null;index" json:"usuario_id"`
	Usuario   *Usuario `gorm:"foreignKey:UsuarioID" json:"usuario,omitempty"`

	DescargoID string    `gorm:"type:uuid;not null;index" json:"descargo_id"`
	Descargo   *Descargo `gorm:"foreignKey:DescargoID" json:"descargo,omitempty"`

	PasajeID *string `gorm:"size:36;index" json:"pasaje_id"`
	Pasaje   *Pasaje `gorm:"foreignKey:PasajeID" json:"pasaje,omitempty"`

	NumeroBillete  string           `gorm:"type:varchar(100);not null;index" json:"numero_billete"`
	TramosNoUsados string           `gorm:"type:text" json:"tramos_no_usados"` // Ej: Pando-Beni, Beni-VVI, VVI-LPB
	Estado         EstadoOpenTicket `gorm:"type:varchar(20);default:'PENDIENTE';index" json:"estado"`

	// Programación de uso
	FechaVueloProgramada *time.Time `json:"fecha_vuelo_programada"`
	RutaProgramada       string     `gorm:"type:text" json:"ruta_programada"`
	AerolineaProgramada  string     `gorm:"type:varchar(100)" json:"aerolinea_programada"`

	Observaciones string `gorm:"type:text" json:"observaciones"`

	CreatedAt time.Time      `gorm:"index;type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`

	CreatedBy *string `gorm:"size:36;default:null"`
	UpdatedBy *string `gorm:"size:36;default:null"`
	DeletedBy *string `gorm:"size:36;default:null"`
}

func (OpenTicket) TableName() string {
	return "open_tickets"
}

func (c OpenTicket) IsDisponible() bool {
	return c.Estado == EstadoOpenTicketDisponible
}

func (c OpenTicket) IsReservado() bool {
	return c.Estado == EstadoOpenTicketReservado
}

func (c OpenTicket) GetEstadoColor() string {
	switch c.Estado {
	case EstadoOpenTicketDisponible:
		return "success"
	case EstadoOpenTicketReservado:
		return "warning"
	case EstadoOpenTicketFinalizado:
		return "neutral"
	case EstadoOpenTicketPendiente:
		return "primary"
	default:
		return "neutral"
	}
}

func (c OpenTicket) IsPendiente() bool {
	return c.Estado == EstadoOpenTicketPendiente
}

func (c OpenTicket) IsFinalizado() bool {
	return c.Estado == EstadoOpenTicketFinalizado
}

func (c OpenTicket) GetFechaOriginalFormat() string {
	if c.Pasaje == nil {
		return "-"
	}
	return c.Pasaje.FechaVuelo.Format("02/01/2006")
}
