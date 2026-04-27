package models

import (
	"fmt"
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
	MontoCredito   float64          `gorm:"type:decimal(10,2);default:0" json:"monto_credito"`
	Estado         EstadoOpenTicket `gorm:"type:varchar(20);default:'PENDIENTE';index" json:"estado"`

	// Uso y Consumo
	SolicitudConsumoID *string    `gorm:"size:36;index" json:"solicitud_consumo_id"`
	SolicitudConsumo   *Solicitud `gorm:"foreignKey:SolicitudConsumoID" json:"solicitud_consumo,omitempty"`

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

func (c OpenTicket) GetDescargoURL() string {
	if c.Descargo == nil {
		return "#"
	}
	tipo := "derecho"
	if c.Descargo.Solicitud != nil && c.Descargo.Solicitud.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/descargos/%s/%s", tipo, c.DescargoID)
}

func (c OpenTicket) GetSolicitudConsumoURL() string {
	if c.SolicitudConsumoID == nil {
		return "#"
	}
	tipo := "derecho"
	if c.SolicitudConsumo != nil && c.SolicitudConsumo.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/solicitudes/%s/%s", tipo, *c.SolicitudConsumoID)
}

func (c OpenTicket) GetSolicitudOriginalURL() string {
	if c.Descargo == nil || c.Descargo.Solicitud == nil {
		return "#"
	}
	tipo := "derecho"
	if c.Descargo.Solicitud.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/solicitudes/%s/%s/detalle", tipo, c.Descargo.SolicitudID)
}
