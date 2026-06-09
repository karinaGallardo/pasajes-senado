package models

import "time"

type EstadoDescargo string

const (
	EstadoDescargoBorrador     EstadoDescargo = "BORRADOR"
	EstadoDescargoEnRevision   EstadoDescargo = "EN_REVISION"
	EstadoDescargoRechazado    EstadoDescargo = "RECHAZADO"
	EstadoDescargoOpenTicket   EstadoDescargo = "OPEN_TICKET"
	EstadoDescargoEnRevisionOT EstadoDescargo = "EN_REVISION_OT"
	EstadoDescargoRechazadoOT  EstadoDescargo = "RECHAZADO_OT"
	EstadoDescargoFinalizado   EstadoDescargo = "FINALIZADO"
)

type DescargoPermissions struct {
	CanEdit               bool
	CanSubmit             bool
	CanApprove            bool
	CanReject             bool
	CanRevert             bool
	CanPrint              bool
	CanCompleteOpenTicket bool
	RevertLabel           string
	ApproveLabel          string
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

	Tramos []DescargoTramo `gorm:"foreignKey:DescargoID"`

	Estado  EstadoDescargo   `gorm:"size:50;default:'BORRADOR'"`
	Oficial *DescargoOficial `gorm:"foreignKey:DescargoID"`

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
	hasItinerary := false
	for _, it := range d.Tramos {
		if !it.EsOpenTicket {
			hasItinerary = true
			if it.Billete == "" || it.NumeroPaseAbordo == "" || it.ArchivoPaseAbordo == "" {
				return false
			}
		}
	}

	if !hasItinerary && !d.allDevolucion() {
		return false
	}

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

func (Descargo) TableName() string {
	return "descargos"
}

func (d Descargo) GetEstadoColorClass() string {
	return EstadoDescargoStatusInfo(d.Estado).ColorClass
}

func (d Descargo) GetEstadoLabel() string {
	return EstadoDescargoStatusInfo(d.Estado).Nombre
}

func (d Descargo) GetEstadoBadgeClass() string {
	return EstadoDescargoStatusInfo(d.Estado).BadgeClass
}

func (d Descargo) GetEstadoIcon() string {
	return EstadoDescargoStatusInfo(d.Estado).Icon
}

func (d Descargo) GetEstadoDescripcion() string {
	return EstadoDescargoStatusInfo(d.Estado).Descripcion
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

	revertLabel := "Revertir a Borrador"

	approveLabel := "Aprobar y Finalizar"

	return DescargoPermissions{
		CanEdit:               canMod,
		CanSubmit:             canMod,
		CanApprove:            d.CanApprove(user),
		CanReject:             d.CanReject(user),
		CanRevert:             d.CanRevert(user),
		CanPrint:              d.Estado != EstadoDescargoBorrador,
		CanCompleteOpenTicket: d.Estado == EstadoDescargoOpenTicket && d.isOwnerOrAdmin(user),
		RevertLabel:           revertLabel,
		ApproveLabel:          approveLabel,
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
	return d.Estado == EstadoDescargoBorrador || d.Estado == EstadoDescargoRechazado || d.Estado == EstadoDescargoRechazadoOT || d.Estado == EstadoDescargoOpenTicket
}

func (d Descargo) IsInRevision() bool {
	return d.Estado == EstadoDescargoEnRevision || d.Estado == EstadoDescargoEnRevisionOT
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
	return (d.Estado == EstadoDescargoEnRevision || d.Estado == EstadoDescargoEnRevisionOT) && user.IsAdminOrResponsable()
}

func (d Descargo) CanReject(user *Usuario) bool {
	return (d.Estado == EstadoDescargoEnRevision || d.Estado == EstadoDescargoEnRevisionOT) && user.IsAdminOrResponsable()
}

func (d Descargo) CanRevert(user *Usuario) bool {
	return (d.Estado == EstadoDescargoFinalizado || d.Estado == EstadoDescargoOpenTicket || d.Estado == EstadoDescargoRechazadoOT) && user.IsAdminOrResponsable()
}

func (d Descargo) CanPrint(user *Usuario) bool {
	return d.Estado != EstadoDescargoBorrador
}

func (d Descargo) ShouldShowFinancialSection() bool {
	return d.GetTotalDevolucionPasajes() > 0
}

func (d Descargo) ShouldShowPaymentProof() bool {
	if d.GetTotalDevolucionPasajes() <= 0 {
		return false
	}
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
	if user.IsAdminOrResponsable() {
		return true
	}
	if d.Solicitud != nil {
		if d.Solicitud.UsuarioID == user.ID {
			return true
		}
		if d.Solicitud.Usuario.ID != "" && d.Solicitud.Usuario.EncargadoID != nil && *d.Solicitud.Usuario.EncargadoID == user.ID {
			return true
		}
	}

	if d.UsuarioID == user.ID {
		return true
	}
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

func (d Descargo) HasOpenTicket() bool {
	for _, t := range d.Tramos {
		if t.EsOpenTicket {
			return true
		}
	}
	return false
}

func (d Descargo) IsOpenTicket() bool {
	return d.Estado == EstadoDescargoOpenTicket
}
