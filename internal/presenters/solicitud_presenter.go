package presenters

import (
	"fmt"
	"sistema-pasajes/internal/models"
)

type SolicitudPresenter struct {
	models.Solicitud
}

func NewSolicitudPresenter(s models.Solicitud) *SolicitudPresenter {
	return &SolicitudPresenter{Solicitud: s}
}

func WrapSolicitudes(list []models.Solicitud) []*SolicitudPresenter {
	presenters := make([]*SolicitudPresenter, len(list))
	for i, s := range list {
		presenters[i] = NewSolicitudPresenter(s)
	}
	return presenters
}

func (p *SolicitudPresenter) StatusBadgeClass() string {
	switch p.GetEstado() {
	case "SOLICITADO":
		return "bg-blue-100 text-blue-800 border-blue-200"
	case "APROBADO":
		return "bg-green-100 text-green-800 border-green-200"
	case "EMITIDO":
		return "bg-purple-100 text-purple-800 border-purple-200"
	case "RECHAZADO":
		return "bg-red-100 text-red-800 border-red-200"
	case "FINALIZADO":
		return "bg-gray-100 text-gray-800 border-gray-200"
	case "PARCIALMENTE_APROBADO":
		return "bg-yellow-100 text-yellow-800 border-yellow-200"
	default:
		return "bg-gray-100 text-gray-800 border-gray-200"
	}
}

func (p *SolicitudPresenter) ConceptoBadgeClass() string {
	switch p.GetConceptoCodigo() {
	case "DERECHO":
		return "bg-indigo-100 text-indigo-700"
	case "OFICIAL":
		return "bg-amber-100 text-amber-700"
	default:
		return "bg-slate-100 text-slate-700"
	}
}

func (p *SolicitudPresenter) DescargoStatus() (string, string) {
	if p.Descargo != nil {
		return "PRESENTADO", "bg-emerald-100 text-emerald-800"
	}

	dias := p.GetDiasRestantesDescargo()
	if dias == 999 { // No emitido aún
		return "PENDIENTE EMISION", "bg-gray-100 text-gray-500"
	}

	if dias < 0 {
		return fmt.Sprintf("VENCIDO (%d d)", -dias), "bg-orange-100 text-orange-800 animate-pulse"
	}

	if dias <= 2 {
		return fmt.Sprintf("POR VENCER (%d d)", dias), "bg-yellow-100 text-yellow-800"
	}

	return fmt.Sprintf("PENDIENTE (%d d)", dias), "bg-sky-100 text-sky-800"
}
