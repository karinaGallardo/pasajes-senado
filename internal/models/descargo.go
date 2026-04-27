package models

import "time"

type EstadoDescargo string

const (
	EstadoDescargoBorrador   EstadoDescargo = "BORRADOR"
	EstadoDescargoEnRevision EstadoDescargo = "EN_REVISION"
	EstadoDescargoRechazado  EstadoDescargo = "RECHAZADO"
	EstadoDescargoOpenTicket EstadoDescargo = "OPEN_TICKET"
	EstadoDescargoFinalizado EstadoDescargo = "FINALIZADO"
)

type DescargoPermissions struct {
	CanEdit    bool
	CanSubmit  bool
	CanApprove bool
	CanReject  bool
	CanRevert  bool
	CanPrint   bool
}

type Descargo struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	UsuarioID   string     `gorm:"size:36;not null"`

	Codigo     string `gorm:"size:20;uniqueIndex"`
	NumeroCite string `gorm:"size:50;index"`

	FechaPresentacion time.Time `gorm:"not null;type:timestamp"`
	Observaciones     string    `gorm:"type:text"`

	// Tramos de Viaje (FV-05) - Relación granular por conexiones
	Tramos []DescargoTramo `gorm:"foreignKey:DescargoID"`

	Estado EstadoDescargo `gorm:"size:50;default:'BORRADOR'"`

	// Detalle opcional para informes oficiales (PV-06)
	Oficial *DescargoOficial `gorm:"foreignKey:DescargoID"`

	// Contexto de runtime (no persistido)
	authUser    *Usuario             `gorm:"-"`
	Permissions *DescargoPermissions `gorm:"-"`
}

func (d Descargo) HasChanges(other Descargo) bool {
	return d.SolicitudID != other.SolicitudID ||
		d.UsuarioID != other.UsuarioID ||
		d.Codigo != other.Codigo ||
		d.NumeroCite != other.NumeroCite ||
		!d.FechaPresentacion.Equal(other.FechaPresentacion) ||
		d.Observaciones != other.Observaciones ||
		d.Estado != other.Estado
}

func (d Descargo) IsComplete() bool {
	// 1. Validar Itinerario (Billete, Pase, Archivo)
	// Solo validamos los que no son devoluciones
	hasItinerary := false
	for _, it := range d.Tramos {
		if !it.EsOpenTicket {
			hasItinerary = true
			if it.Billete == "" || it.NumeroPaseAbordo == "" || it.ArchivoPaseAbordo == "" {
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

func (d Descargo) GetMissingItemsHTML() string {
	var missing []string

	// Check Itinerary
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

	if !hasItinerary && !d.allDevolucion() {
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
	case EstadoDescargoOpenTicket:
		return EstadoDescargoInfo{
			Nombre:      "En Espera (Reprogramación)",
			Descripcion: "El descargo tiene tramos pendientes de reprogramar (Open Ticket). Debe re-editarse para agregar los nuevos vuelos.",
			ColorClass:  "border-secondary-400",
			BadgeClass:  "bg-secondary-50 text-secondary-700 border-secondary-100",
			Icon:        "ph ph-calendar-plus",
		}
	case EstadoDescargoFinalizado:
		return EstadoDescargoInfo{
			Nombre:      "Finalizado",
			Descripcion: "El descargo ha sido validado y aprobado satisfactoriamente.",
			ColorClass:  "border-neutral-900",
			BadgeClass:  "bg-neutral-900 text-white border-neutral-800",
			Icon:        "ph ph-flag-checkered",
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

func (d Descargo) GetConcepto() string {
	if d.Solicitud != nil {
		return d.Solicitud.GetConcepto()
	}
	return "derecho"
}

func (d Descargo) GetPermissions(u ...*Usuario) DescargoPermissions {
	user := d.getAuthUser(u...)
	if user == nil {
		return DescargoPermissions{}
	}

	canMod := d.IsEditable() && d.isOwnerOrAdmin(user)

	return DescargoPermissions{
		CanEdit:    canMod,
		CanSubmit:  canMod,
		CanApprove: d.CanApprove(user),
		CanReject:  d.CanReject(user),
		CanRevert:  d.CanRevert(user),
		CanPrint:   d.Estado != EstadoDescargoBorrador,
	}
}

func (d *Descargo) HydratePermissions(u ...*Usuario) {
	if len(u) > 0 {
		d.authUser = u[0]
	}
	p := d.GetPermissions()
	d.Permissions = &p
}

func (d Descargo) getAuthUser(u ...*Usuario) *Usuario {
	if len(u) > 0 {
		return u[0]
	}
	return d.authUser
}

func (d Descargo) IsEditable() bool {
	return d.Estado == EstadoDescargoBorrador || d.Estado == EstadoDescargoRechazado || d.Estado == EstadoDescargoOpenTicket
}

func (d Descargo) CanEdit(user *Usuario) bool {
	if !d.IsEditable() {
		return false
	}
	return d.isOwnerOrAdmin(user)
}

func (d Descargo) CanSubmit(user *Usuario) bool {
	if !d.IsEditable() {
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
	return (d.Estado == EstadoDescargoFinalizado || d.Estado == EstadoDescargoOpenTicket) && user.IsAdminOrResponsable()
}

func (d Descargo) CanPrint(user *Usuario) bool {
	// Solo se puede imprimir si no está en borrador o si el usuario tiene privilegios para ver previsualizaciones
	return d.Estado != EstadoDescargoBorrador
}

func (d Descargo) ShouldShowFinancialSection() bool {
	return d.GetTotalDevolucionPasajes() > 0
}

func (d Descargo) ShouldShowPaymentProof() bool {
	// Solo si hay un monto real que devolver
	if d.GetTotalDevolucionPasajes() <= 0 {
		return false
	}
	// Se presume que si está FINALIZADO, al menos un pasaje tiene el archivo
	return d.Estado == EstadoDescargoFinalizado
}

func (d Descargo) CanRevertFinalization(u *Usuario) bool {
	if u == nil {
		return false
	}
	return u.IsAdminOrResponsable() && (d.Estado == EstadoDescargoFinalizado || d.Estado == EstadoDescargoOpenTicket)
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
		if d.Solicitud.Usuario.ID != "" && d.Solicitud.Usuario.EncargadoID != nil && *d.Solicitud.Usuario.EncargadoID == user.ID {
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

func (d Descargo) GetTotalDevolucionPasajes() float64 {
	totalValue := 0.0
	if d.Solicitud != nil {
		for _, item := range d.Solicitud.Items {
			for _, p := range item.Pasajes {
				totalValue += p.MontoReembolso
			}
		}
	}
	return totalValue
}
