package models

import "time"

type EstadoDescargo string

const (
	EstadoDescargoBorrador   EstadoDescargo = "BORRADOR"
	EstadoDescargoEnRevision EstadoDescargo = "EN_REVISION"
	EstadoDescargoRechazado  EstadoDescargo = "RECHAZADO"
	EstadoDescargoAprobado   EstadoDescargo = "APROBADO"
)

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

	Estado EstadoDescargo `gorm:"size:50;default:'BORRADOR'"`

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

type EstadoDescargoInfo struct {
	Nombre      string
	Descripcion string
	ColorClass  string
	BadgeClass  string
	Icon        string
}

func (e EstadoDescargo) Info() EstadoDescargoInfo {
	switch e {
	case EstadoDescargoBorrador:
		return EstadoDescargoInfo{
			Nombre:      "Borrador",
			Descripcion: "El descargo se encuentra en etapa de preparación y edición por el beneficiario.",
			ColorClass:  "border-neutral-400",
			BadgeClass:  "bg-neutral-50 text-neutral-600 border-neutral-100",
			Icon:        "ph ph-pencil-line",
		}
	case EstadoDescargoEnRevision:
		return EstadoDescargoInfo{
			Nombre:      "En Revisión",
			Descripcion: "El descargo ha sido enviado y está siendo revisado por el área administrativa.",
			ColorClass:  "border-warning-400",
			BadgeClass:  "bg-warning-50 text-warning-700 border-warning-100",
			Icon:        "ph ph-clock",
		}
	case EstadoDescargoRechazado:
		return EstadoDescargoInfo{
			Nombre:      "Observado",
			Descripcion: "Se han encontrado observaciones en el descargo. Requiere corrección y reenvío.",
			ColorClass:  "border-danger-400",
			BadgeClass:  "bg-danger-50 text-danger-700 border-danger-100",
			Icon:        "ph ph-warning-circle",
		}
	case EstadoDescargoAprobado:
		return EstadoDescargoInfo{
			Nombre:      "Aprobado",
			Descripcion: "El descargo ha sido validado y aprobado satisfactoriamente.",
			ColorClass:  "border-success-400",
			BadgeClass:  "bg-success-50 text-success-700 border-success-100",
			Icon:        "ph ph-check-circle",
		}
	default:
		return EstadoDescargoInfo{
			Nombre:      string(e),
			Descripcion: "-",
			ColorClass:  "border-neutral-200",
			BadgeClass:  "bg-neutral-50 text-neutral-700",
			Icon:        "ph ph-info",
		}
	}
}

func (Descargo) TableName() string {
	return "descargos"
}

// --- Display Helpers (Refactored to use Enum Info) ---

func (d Descargo) GetEstadoColorClass() string {
	return d.Estado.Info().ColorClass
}

func (d Descargo) GetEstadoLabel() string {
	return d.Estado.Info().Nombre
}

func (d Descargo) GetEstadoBadgeClass() string {
	return d.Estado.Info().BadgeClass
}

func (d Descargo) GetEstadoIcon() string {
	return d.Estado.Info().Icon
}

func (d Descargo) GetEstadoDescripcion() string {
	return d.Estado.Info().Descripcion
}

// --- Permission Helpers ---

func (d Descargo) CanEdit(user *Usuario) bool {
	if d.Estado != EstadoDescargoBorrador && d.Estado != EstadoDescargoRechazado {
		return false
	}
	return d.isOwnerOrAdmin(user)
}

func (d Descargo) CanSubmit(user *Usuario) bool {
	if d.Estado != EstadoDescargoBorrador && d.Estado != EstadoDescargoRechazado {
		return false
	}
	return d.isOwnerOrAdmin(user)
}

func (d Descargo) CanApprove(user *Usuario) bool {
	return d.Estado == EstadoDescargoEnRevision && user.IsAdminOrResponsable()
}

func (d Descargo) CanReject(user *Usuario) bool {
	return d.Estado == EstadoDescargoEnRevision && user.IsAdminOrResponsable()
}

func (d Descargo) CanRevert(user *Usuario) bool {
	return d.Estado == EstadoDescargoAprobado && user.IsAdminOrResponsable()
}

func (d Descargo) isOwnerOrAdmin(user *Usuario) bool {
	if user == nil {
		return false
	}
	// 1. Administradores y Responsables de Pasajes
	if user.IsAdminOrResponsable() {
		return true
	}

	// 2. Verificación basada en la Solicitud (Fuente de verdad del viaje)
	if d.Solicitud != nil {
		// El Titular del viaje (Senador/Funcionario beneficiario)
		if d.Solicitud.UsuarioID == user.ID {
			return true
		}
		// El Asistente/Encargado asignado al titular del viaje
		if d.Solicitud.Usuario.EncargadoID != nil && *d.Solicitud.Usuario.EncargadoID == user.ID {
			return true
		}
	}

	// 3. Propietario del registro de Descargo (Fallback)
	if d.UsuarioID == user.ID {
		return true
	}

	// 4. Creador Físico del registro
	if d.CreatedBy != nil && *d.CreatedBy == user.ID {
		return true
	}

	return false
}

// --- Field Helpers ---

func (d Descargo) GetMemorandum() string {
	if d.Oficial != nil && d.Oficial.NroMemorandum != "" {
		return d.Oficial.NroMemorandum
	}
	return " - "
}

func (d Descargo) GetObjetivo() string {
	if d.Oficial != nil && d.Oficial.ObjetivoViaje != "" {
		return d.Oficial.ObjetivoViaje
	}
	if d.Solicitud != nil {
		return d.Solicitud.Motivo
	}
	return " - "
}

func (d Descargo) GetActividades() string {
	if d.Oficial != nil {
		return d.Oficial.InformeActividades
	}
	return ""
}

func (d Descargo) GetResultados() string {
	if d.Oficial != nil {
		return d.Oficial.ResultadosViaje
	}
	return ""
}

func (d Descargo) GetConclusiones() string {
	if d.Oficial != nil {
		return d.Oficial.ConclusionesRecomendaciones
	}
	return ""
}

func (d Descargo) GetTransporteDisplay() string {
	if d.Oficial != nil {
		return d.Oficial.GetTipoTransporteDisplay()
	}
	return " - "
}
