package presenters

import (
	"fmt"
	"sistema-pasajes/internal/models"
)

type OpenTicketPresenter struct {
	Ticket models.OpenTicket
}

func NewOpenTicketPresenter(t models.OpenTicket) *OpenTicketPresenter {
	return &OpenTicketPresenter{Ticket: t}
}

func (p *OpenTicketPresenter) StatusColor() string {
	switch p.Ticket.Estado {
	case models.EstadoOpenTicketDisponible:
		return "success"
	case models.EstadoOpenTicketReservado:
		return "warning"
	case models.EstadoOpenTicketFinalizado:
		return "neutral"
	case models.EstadoOpenTicketPendiente:
		return "primary"
	default:
		return "neutral"
	}
}

func (p *OpenTicketPresenter) StatusBadgeClass() string {
	color := p.StatusColor()
	return fmt.Sprintf("bg-%s-100 text-%s-700 border border-%s-200", color, color, color)
}

func (p *OpenTicketPresenter) StatusLabel() string {
	switch p.Ticket.Estado {
	case models.EstadoOpenTicketPendiente:
		return "Pendiente"
	case models.EstadoOpenTicketDisponible:
		return "Disponible"
	case models.EstadoOpenTicketReservado:
		return "Reservado"
	case models.EstadoOpenTicketFinalizado:
		return "Finalizado"
	case models.EstadoOpenTicketCancelado:
		return "Cancelado"
	default:
		return string(p.Ticket.Estado)
	}
}

func (p *OpenTicketPresenter) FechaOriginal() string {
	if p.Ticket.Pasaje == nil {
		return "-"
	}
	return p.Ticket.Pasaje.FechaVuelo.Format("02/01/2006")
}

func (p *OpenTicketPresenter) DescargoURL() string {
	if p.Ticket.Descargo == nil {
		return "#"
	}
	tipo := "derecho"
	if p.Ticket.Descargo.Solicitud != nil && p.Ticket.Descargo.Solicitud.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/descargos/%s/%s", tipo, p.Ticket.DescargoID)
}

func (p *OpenTicketPresenter) SolicitudConsumoURL() string {
	if p.Ticket.SolicitudConsumoID == nil {
		return "#"
	}
	tipo := "derecho"
	if p.Ticket.SolicitudConsumo != nil && p.Ticket.SolicitudConsumo.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/solicitudes/%s/%s", tipo, *p.Ticket.SolicitudConsumoID)
}

func (p *OpenTicketPresenter) SolicitudOriginalURL() string {
	if p.Ticket.Descargo == nil || p.Ticket.Descargo.Solicitud == nil {
		return "#"
	}
	tipo := "derecho"
	if p.Ticket.Descargo.Solicitud.IsOficial() {
		tipo = "oficial"
	}
	return fmt.Sprintf("/solicitudes/%s/%s/detalle", tipo, p.Ticket.Descargo.SolicitudID)
}
