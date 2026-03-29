package models

import (
	"strings"
	"time"
)

type Solicitud struct {
	BaseModel
	Codigo    string  `gorm:"size:12;uniqueIndex"`
	UsuarioID string  `gorm:"size:24;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	CupoDerechoItemID *string          `gorm:"size:36;index;default:null"`
	CupoDerechoItem   *CupoDerechoItem `gorm:"foreignKey:CupoDerechoItemID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	TipoSolicitudCodigo string         `gorm:"size:50;not null;index"`
	TipoSolicitud       *TipoSolicitud `gorm:"foreignKey:TipoSolicitudCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	AmbitoViajeCodigo string       `gorm:"size:20;not null;index"`
	AmbitoViaje       *AmbitoViaje `gorm:"foreignKey:AmbitoViajeCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	TipoItinerarioCodigo string          `gorm:"size:20;not null;index"`
	TipoItinerario       *TipoItinerario `gorm:"foreignKey:TipoItinerarioCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	EstadoSolicitudCodigo *string          `gorm:"size:50;index;default:'SOLICITADO'"`
	EstadoSolicitud       *EstadoSolicitud `gorm:"foreignKey:EstadoSolicitudCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	Viaticos []Viatico `gorm:"foreignKey:SolicitudID"`

	Descargo *Descargo `gorm:"foreignKey:SolicitudID"`

	Autorizacion string `gorm:"size:100;index"`

	Motivo string `gorm:"type:text"`

	AerolineaSugerida string `gorm:"size:100;comment:Aerolinea sugerida para todos los tramos"`

	// New Decoupled Items
	Items []SolicitudItem `gorm:"foreignKey:SolicitudID"`
}

func (Solicitud) TableName() string {
	return "solicitudes"
}

func (s Solicitud) GetEstado() string {
	if s.EstadoSolicitudCodigo == nil {
		return "SOLICITADO"
	}
	return *s.EstadoSolicitudCodigo
}

func (s Solicitud) GetEstadoCodigo() string {
	if s.EstadoSolicitudCodigo == nil {
		return ""
	}
	return *s.EstadoSolicitudCodigo
}

func (s Solicitud) GetConceptoNombre() string {
	if s.TipoSolicitud != nil && s.TipoSolicitud.ConceptoViaje != nil {
		return s.TipoSolicitud.ConceptoViaje.Nombre
	}
	return ""
}

func (s Solicitud) GetTipoNombre() string {
	if s.TipoSolicitud != nil {
		return s.TipoSolicitud.Nombre
	}
	return ""
}

func (s Solicitud) GetConceptoCodigo() string {
	if s.TipoSolicitud != nil && s.TipoSolicitud.ConceptoViaje != nil {
		return s.TipoSolicitud.ConceptoViaje.Codigo
	}
	return ""
}

func (s *Solicitud) UpdateStatusBasedOnItems() {
	if len(s.Items) == 0 {
		return
	}

	hasIda := false
	hasVuelta := false
	allApproved := true
	allRejected := true
	allFinalized := true
	allEmitidos := true
	hasApproved := false

	for _, item := range s.Items {
		st := item.GetEstado()
		if item.Tipo == TipoSolicitudItemIda {
			hasIda = true
		}
		if item.Tipo == TipoSolicitudItemVuelta {
			hasVuelta = true
		}

		// Consideramos estados de aprobación (Aprobado, Emitido, Finalizado)
		isApp := (st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO")

		if !isApp {
			allApproved = false
		} else {
			hasApproved = true
		}

		if st != "FINALIZADO" {
			allFinalized = false
		}

		if st != "EMITIDO" && st != "FINALIZADO" {
			allEmitidos = false
		}

		if st != "RECHAZADO" {
			allRejected = false
		}
	}

	// Update Itinerary Type Informatively (if it was a Derecho)
	// Generally applicable: if has both, IDA_VUELTA. If only IDA, SOLO_IDA. If only VUELTA, SOLO_VUELTA.
	if hasIda && hasVuelta {
		s.TipoItinerarioCodigo = "IDA_VUELTA"
	} else if hasIda {
		s.TipoItinerarioCodigo = "SOLO_IDA"
	} else if hasVuelta {
		s.TipoItinerarioCodigo = "SOLO_VUELTA"
	}

	// Unificamos criterio: Para estar completamente Aprobado/Emitido/Finalizado,
	// debe tener ambos tramos (Ida y Vuelta). Si falta uno, es Parcial.
	newState := "SOLICITADO"

	if allFinalized && hasIda && hasVuelta {
		newState = "FINALIZADO"
	} else if allEmitidos && hasIda && hasVuelta {
		newState = "EMITIDO"
	} else if allApproved && hasIda && hasVuelta {
		newState = "APROBADO"
	} else if hasApproved || (allApproved && (!hasIda || !hasVuelta)) || (allEmitidos && (!hasIda || !hasVuelta)) {
		newState = "PARCIALMENTE_APROBADO"
	} else if allRejected {
		newState = "RECHAZADO"
	} else {
		newState = "SOLICITADO"
	}

	s.EstadoSolicitudCodigo = &newState
}

func (s Solicitud) GetFechaIda() *time.Time {
	for i := range s.Items {
		item := &s.Items[i]
		if item.Tipo == TipoSolicitudItemIda && item.Fecha != nil {
			return item.Fecha
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Fecha
	}
	return nil
}

func (s Solicitud) GetFechaVuelta() *time.Time {
	for i := range s.Items {
		item := &s.Items[i]
		if item.Tipo == TipoSolicitudItemVuelta && item.Fecha != nil {
			return item.Fecha
		}
	}
	if len(s.Items) > 1 {
		return s.Items[len(s.Items)-1].Fecha
	}
	return nil
}

func (s Solicitud) GetOrigen() *Destino {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return s.Items[i].Origen
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Origen
	}
	return nil
}

func (s Solicitud) GetDestino() *Destino {
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return s.Items[i].Destino
		}
	}
	if len(s.Items) > 0 {
		return s.Items[0].Destino
	}
	return nil
}

func (s Solicitud) GetOrigenCiudad() string {
	obj := s.GetOrigen()
	if obj != nil {
		return obj.Ciudad
	}
	return "-"
}

func (s Solicitud) GetDestinoCiudad() string {
	obj := s.GetDestino()
	if obj != nil {
		return obj.Ciudad
	}
	return "-"
}

func (s Solicitud) GetOrigenIATA() string {
	obj := s.GetOrigen()
	if obj != nil {
		return obj.IATA
	}
	return ""
}

func (s Solicitud) GetDestinoIATA() string {
	obj := s.GetDestino()
	if obj != nil {
		return obj.IATA
	}
	return ""
}

func (s Solicitud) GetRutaSimple() string {
	origen := s.GetOrigenIATA()
	destino := s.GetDestinoIATA()
	if origen == "" || destino == "" {
		return s.GetOrigenCiudad() + " - " + s.GetDestinoCiudad()
	}
	return origen + " - " + destino
}

func (s Solicitud) GetItemIda() *SolicitudItem {
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "IDA" {
			return &s.Items[i]
		}
	}
	return nil
}

func (s Solicitud) GetItemVuelta() *SolicitudItem {
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "VUELTA" {
			return &s.Items[i]
		}
	}
	return nil
}

func (s Solicitud) GetMaxFechaVueloEmitida() *time.Time {
	var maxDate *time.Time
	for _, item := range s.Items {
		for _, p := range item.Pasajes {
			if p.GetEstadoCodigo() == "EMITIDO" {
				if maxDate == nil || p.FechaVuelo.After(*maxDate) {
					fecha := p.FechaVuelo
					maxDate = &fecha
				}
			}
		}
	}
	return maxDate
}

func (s Solicitud) GetUltimoVueloFecha() string {
	maxDate := s.GetMaxFechaVueloEmitida()
	if maxDate == nil {
		return "-"
	}
	return maxDate.Format("02/01/2006")
}

// GetDiasRestantesDescargo calcula cuántos días hábiles le quedan para presentar.
// Retorna un número negativo si ya venció.
func (s Solicitud) GetDiasRestantesDescargo() int {
	maxDate := s.GetMaxFechaVueloEmitida()
	if maxDate == nil {
		return 999
	}

	// 1. Obtener fecha límite (8 días hábiles desde el último vuelo)
	// Como no puedo importar utils aquí directamente (circular dependency),
	// haré la lógica simple o asumiré que se calcula fuera.
	// Pero mejor la pongo aquí si puedo o uso una función auxiliar.
	// Nota: No puedo importar utils. Calculemos aquí.

	diasLimite := 0
	limite := *maxDate
	for diasLimite < 8 {
		limite = limite.AddDate(0, 0, 1)
		if limite.Weekday() != time.Saturday && limite.Weekday() != time.Sunday {
			diasLimite++
		}
	}

	// 2. Contar días hábiles desde HOY hasta el límite
	hoy := time.Now().Truncate(24 * time.Hour)
	limiteTrunc := limite.Truncate(24 * time.Hour)

	if hoy.After(limiteTrunc) {
		// Calcular cuántos días hábiles de mora
		mora := 0
		d := limiteTrunc
		for d.Before(hoy) {
			d = d.AddDate(0, 0, 1)
			if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
				mora++
			}
		}
		return -mora
	}

	// Días restantes
	restantes := 0
	d := hoy
	for d.Before(limiteTrunc) {
		d = d.AddDate(0, 0, 1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			restantes++
		}
	}
	return restantes
}

// --- Authorization Logic ---

func (s Solicitud) CanView(user *Usuario) bool {
	if user.IsAdminOrResponsable() {
		return true
	}
	if s.UsuarioID == user.ID {
		return true
	}
	if s.CreatedBy != nil && *s.CreatedBy == user.ID {
		return true
	}
	if s.Usuario.EncargadoID != nil && *s.Usuario.EncargadoID == user.ID {
		return true
	}
	return false
}

func (s Solicitud) CanEdit(user *Usuario) bool {
	estado := s.GetEstado()
	isEditableState := (estado == "SOLICITADO" || estado == "RECHAZADO" || estado == "PARCIALMENTE_APROBADO")
	if !isEditableState {
		return false
	}
	return s.CanView(user)
}

func (s Solicitud) CanDelete(user *Usuario) bool {
	estado := s.GetEstado()
	// Only creators/admins can delete and only in SOLICITADO state
	if estado != "SOLICITADO" {
		return false
	}
	if user.IsAdminOrResponsable() {
		return true
	}
	if s.CreatedBy != nil && *s.CreatedBy == user.ID {
		return true
	}
	return false
}

func (s Solicitud) CanApprove(user *Usuario) bool {
	if !user.CanApproveReject() {
		return false
	}
	estado := s.GetEstado()
	return estado == "SOLICITADO" || estado == "PARCIALMENTE_APROBADO"
}

func (s Solicitud) CanReject(user *Usuario) bool {
	return s.CanApprove(user)
}

func (s Solicitud) CanAssignPasaje(user *Usuario) bool {
	// Solo Responsables o Admins pueden asignar pasajes electrónicos
	return user.IsAdminOrResponsable()
}

func (s Solicitud) CanAssignViatico(user *Usuario) bool {
	return user.IsAdminOrResponsable()
}

func (s Solicitud) HasEmittedPasaje() bool {
	for _, item := range s.Items {
		for _, p := range item.Pasajes {
			if p.GetEstadoCodigo() == "EMITIDO" || p.GetEstadoCodigo() == "USADO" {
				return true
			}
		}
	}
	return false
}

func (s Solicitud) CanRevertApproval(user *Usuario) bool {
	if !user.IsAdminOrResponsable() {
		return false
	}
	st := s.GetEstado()
	canRevertState := st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO"
	return canRevertState && !s.HasEmittedPasaje()
}

func (s Solicitud) CanMakeDescargo(user *Usuario) bool {
	return s.HasEmittedPasaje() && s.CanView(user)
}

func (s Solicitud) CanPrint(user *Usuario) bool {
	// Solo se imprime si está aprobada o emitida
	st := s.GetEstado()
	canPrintState := st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
	return canPrintState && s.CanView(user)
}

// --- Display Helpers ---

func (s Solicitud) IsDerecho() bool {
	return s.GetConceptoCodigo() == "DERECHO"
}

func (s Solicitud) IsOficial() bool {
	return s.GetConceptoCodigo() == "OFICIAL"
}

func (s Solicitud) GetStatusBadgeClass() string {
	switch s.GetEstado() {
	case "SOLICITADO":
		return "bg-primary-100 text-primary-700"
	case "RECHAZADO":
		return "bg-danger-100 text-danger-700"
	case "APROBADO":
		return "bg-success-100 text-success-700 font-bold"
	case "PARCIALMENTE_APROBADO":
		return "bg-violet-100 text-violet-700"
	case "EMITIDO":
		return "bg-secondary-100 text-secondary-700 font-bold"
	case "FINALIZADO":
		return "bg-neutral-800 text-white"
	default:
		return "bg-neutral-100 text-neutral-600"
	}
}

func (s Solicitud) GetStepNumber() int {
	switch s.GetEstado() {
	case "SOLICITADO", "RECHAZADO":
		return 1
	case "APROBADO", "PARCIALMENTE_APROBADO":
		return 2
	case "EMITIDO":
		return 3
	case "FINALIZADO":
		return 4
	default:
		return 1
	}
}

func (s Solicitud) HasCompleteDescargo() bool {
	if s.Descargo == nil {
		return false
	}

	// 1. Validar Itinerario (Boleto, Pase, Archivo)
	hasItinerary := false
	for _, it := range s.Descargo.DetallesItinerario {
		if !it.EsDevolucion {
			hasItinerary = true
			if it.Boleto == "" || it.NumeroPaseAbordo == "" || it.ArchivoPaseAbordo == "" {
				return false
			}
		}
	}

	// Si no tiene itinerario registrado, lo consideramos incompleto (a menos que todo sea devolución)
	if !hasItinerary && !s.Descargo.allDevolucion() {
		return false
	}

	// 2. Validar Informe Oficial (si aplica)
	if s.GetConceptoCodigo() == "OFICIAL" {
		if s.Descargo.Oficial == nil {
			return false
		}
		if s.Descargo.Oficial.ObjetivoViaje == "" ||
			s.Descargo.Oficial.InformeActividades == "" ||
			s.Descargo.Oficial.ResultadosViaje == "" ||
			s.Descargo.Oficial.ConclusionesRecomendaciones == "" {
			return false
		}
	}

	return true
}

func (s Solicitud) GetDescargoMissingItems() string {
	if s.Descargo == nil {
		return "No se ha iniciado el descargo"
	}

	var missing []string

	// Check Itinerary
	hasItinerary := false
	missingBoletos := false
	missingPases := false
	missingArchivos := false

	for _, it := range s.Descargo.DetallesItinerario {
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

	if !hasItinerary && !s.Descargo.allDevolucion() {
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
	if s.GetConceptoCodigo() == "OFICIAL" {
		if s.Descargo.Oficial == nil {
			missing = append(missing, "Informe Oficial PV-06")
		} else {
			if s.Descargo.Oficial.ObjetivoViaje == "" ||
				s.Descargo.Oficial.InformeActividades == "" ||
				s.Descargo.Oficial.ResultadosViaje == "" ||
				s.Descargo.Oficial.ConclusionesRecomendaciones == "" {
				missing = append(missing, "Campos del Informe Detallado")
			}
		}
	}

	if len(missing) == 0 {
		return "Datos completos"
	}

	var html strings.Builder
	html.WriteString("<div class='text-left'><p class='text-[10px] font-bold border-b border-white/20 pb-1 mb-1'>Pendiente de llenar:</p><ul class='list-inside list-disc space-y-0.5'>")
	for _, item := range missing {
		html.WriteString("<li class='text-[10px]'>" + item + "</li>")
	}
	html.WriteString("</ul></div>")

	return html.String()
}

// GetChanges compares current solicitud with old state and returns dirty fields map for GORM Updates
func (s *Solicitud) GetChanges(old Solicitud) map[string]any {
	changes := make(map[string]any)

	if s.TipoSolicitudCodigo != old.TipoSolicitudCodigo {
		changes["tipo_solicitud_codigo"] = s.TipoSolicitudCodigo
	}
	if s.AmbitoViajeCodigo != old.AmbitoViajeCodigo {
		changes["ambito_viaje_codigo"] = s.AmbitoViajeCodigo
	}
	if s.AerolineaSugerida != old.AerolineaSugerida {
		changes["aerolinea_sugerida"] = s.AerolineaSugerida
	}
	if s.Motivo != old.Motivo {
		changes["motivo"] = s.Motivo
	}

	// Comparar estado del padre
	if (s.EstadoSolicitudCodigo == nil) != (old.EstadoSolicitudCodigo == nil) ||
		(s.EstadoSolicitudCodigo != nil && old.EstadoSolicitudCodigo != nil && *s.EstadoSolicitudCodigo != *old.EstadoSolicitudCodigo) {
		changes["estado_solicitud_codigo"] = s.EstadoSolicitudCodigo
	}

	return changes
}
