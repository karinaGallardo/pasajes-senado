package presenters

import (
	"fmt"
	"sistema-pasajes/internal/models"
	"strings"
)

type DescargoPresenter struct {
	Descargo models.Descargo
}

func NewDescargoPresenter(d models.Descargo) *DescargoPresenter {
	return &DescargoPresenter{Descargo: d}
}

func WrapDescargos(list []models.Descargo) []*DescargoPresenter {
	presenters := make([]*DescargoPresenter, len(list))
	for i, d := range list {
		presenters[i] = NewDescargoPresenter(d)
	}
	return presenters
}

func (p *DescargoPresenter) StatusColorClass() string {
	return models.EstadoDescargoStatusInfo(p.Descargo.Estado).ColorClass
}

func (p *DescargoPresenter) StatusLabel() string {
	return models.EstadoDescargoStatusInfo(p.Descargo.Estado).Nombre
}

func (p *DescargoPresenter) StatusBadgeClass() string {
	return models.EstadoDescargoStatusInfo(p.Descargo.Estado).BadgeClass
}

func (p *DescargoPresenter) StatusIcon() string {
	return models.EstadoDescargoStatusInfo(p.Descargo.Estado).Icon
}

func (p *DescargoPresenter) StatusDescripcion() string {
	return models.EstadoDescargoStatusInfo(p.Descargo.Estado).Descripcion
}

func (p *DescargoPresenter) GetMissingItemsHTML() string {
	d := p.Descargo
	var missing []string

	hasItinerary := false
	missingBilletes := false
	missingPases := false
	missingArchivos := false

	for _, it := range d.Tramos {
		if !it.EsOpenTicket {
			hasItinerary = true
			if it.Billete == "" {
				missingBilletes = true
			}
			if it.NumeroPaseAbordo == "" {
				missingPases = true
			}
			if it.ArchivoPaseAbordo == "" {
				missingArchivos = true
			}
		}
	}

	if !hasItinerary && !allDevolucion(d) {
		missing = append(missing, "Itinerario de viaje")
	} else {
		if missingBilletes {
			missing = append(missing, "N° Billetes")
		}
		if missingPases {
			missing = append(missing, "N° Pases a Bordo")
		}
		if missingArchivos {
			missing = append(missing, "Archivos PDF/Imagen de Pases")
		}
	}

	if d.Solicitud != nil && d.Solicitud.GetConceptoCodigo() == "OFICIAL" {
		if d.Oficial == nil {
			missing = append(missing, "Informe Oficial PV-06")
		} else {
			if d.Oficial.ObjetivoViaje == "" ||
				d.Oficial.InformeActividades == "" ||
				d.Oficial.ResultadosViaje == "" ||
				d.Oficial.ConclusionesRecomendaciones == "" {
				missing = append(missing, "Campos del Informe Detallado")
			}
		}
	}

	if len(missing) == 0 {
		return "Datos completos"
	}

	var html strings.Builder
	html.WriteString("<div class='text-left'><p class='text-[10px] font-bold border-b border-white/20 pb-1 mb-1'>Pendiente:</p><ul class='list-inside list-disc space-y-0.5'>")
	for _, item := range missing {
		html.WriteString(fmt.Sprintf("<li class='text-[10px]'>%s</li>", item))
	}
	html.WriteString("</ul></div>")

	return html.String()
}

func allDevolucion(d models.Descargo) bool {
	if len(d.Tramos) == 0 {
		return false
	}
	for _, it := range d.Tramos {
		if !it.EsOpenTicket {
			return false
		}
	}
	return true
}
