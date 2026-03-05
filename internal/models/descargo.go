package models

import "time"

type Descargo struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	UsuarioID   string     `gorm:"size:24;not null"`

	Codigo     string `gorm:"size:20;uniqueIndex"`
	NumeroCite string `gorm:"size:50;index"`

	FechaPresentacion time.Time `gorm:"not null;type:timestamp"`
	Observaciones     string    `gorm:"type:text"`

	// Detalles de Itinerario (FV-05) - Relación granular por conexiones
	DetallesItinerario []DetalleItinerarioDescargo `gorm:"foreignKey:DescargoID"`

	Estado string `gorm:"size:50;default:'EN_REVISION'"`

	Documentos []DocumentoDescargo `gorm:"foreignKey:DescargoID"`

	// Detalle opcional para informes oficiales (PV-06)
	Oficial *DescargoOficial `gorm:"foreignKey:DescargoID"`
}

func (d Descargo) IsComplete() bool {
	// 1. Validar Itinerario (Boleto, Pase, Archivo)
	// Solo validamos los que no son devoluciones
	hasItinerary := false
	for _, it := range d.DetallesItinerario {
		if !it.EsDevolucion {
			hasItinerary = true
			if it.Boleto == "" || it.NumeroPaseAbordo == "" || it.ArchivoPaseAbordo == "" {
				return false
			}
		}
	}

	// Si no tiene itinerario registrado, lo consideramos incompleto
	if !hasItinerary && !d.allDevolucion() {
		return false
	}

	// 2. Validar Informe Oficial (si aplica)
	// Si la solicitud es OFICIAL, DEBE tener el reporte oficial lleno
	isOficial := false
	if d.Solicitud != nil {
		isOficial = d.Solicitud.GetConceptoCodigo() == "OFICIAL"
	}

	if isOficial {
		if d.Oficial == nil {
			return false
		}
		if d.Oficial.ObjetivoViaje == "" ||
			d.Oficial.InformeActividades == "" ||
			d.Oficial.ResultadosViaje == "" ||
			d.Oficial.ConclusionesRecomendaciones == "" {
			return false
		}
	}

	return true
}

func (d Descargo) allDevolucion() bool {
	if len(d.DetallesItinerario) == 0 {
		return false
	}
	for _, it := range d.DetallesItinerario {
		if !it.EsDevolucion {
			return false
		}
	}
	return true
}

func (d Descargo) GetMissingItemsHTML() string {
	var missing []string

	// Check Itinerary
	hasItinerary := false
	missingBoletos := false
	missingPases := false
	missingArchivos := false

	for _, it := range d.DetallesItinerario {
		if !it.EsDevolucion {
			hasItinerary = true
			if it.Boleto == "" {
				missingBoletos = true
			}
			if it.NumeroPaseAbordo == "" {
				missingPases = true
			}
			if it.ArchivoPaseAbordo == "" {
				missingArchivos = true
			}
		}
	}

	if !hasItinerary && !d.allDevolucion() {
		missing = append(missing, "Itinerario de viaje")
	} else {
		if missingBoletos {
			missing = append(missing, "N° Boletos")
		}
		if missingPases {
			missing = append(missing, "N° Pases a Bordo")
		}
		if missingArchivos {
			missing = append(missing, "Archivos PDF/Imagen de Pases")
		}
	}

	// Check Official Report
	isOficial := false
	if d.Solicitud != nil {
		isOficial = d.Solicitud.GetConceptoCodigo() == "OFICIAL"
	}

	if isOficial {
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

	html := "<div class='text-left'><p class='text-[10px] font-bold border-b border-white/20 pb-1 mb-1'>Pendiente:</p><ul class='list-inside list-disc space-y-0.5'>"
	for i := range missing {
		html += "<li class='text-[10px]'>" + missing[i] + "</li>"
	}
	html += "</ul></div>"

	return html
}

func (Descargo) TableName() string {
	return "descargos"
}
