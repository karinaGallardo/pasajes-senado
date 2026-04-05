package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
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

	AerolineaID *string    `gorm:"size:36;index;default:null"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	// New Decoupled Items
	Items []SolicitudItem `gorm:"foreignKey:SolicitudID"`

	// Contexto de runtime (no persistido)
	authUser    *Usuario              `gorm:"-"`
	Permissions *SolicitudPermissions `gorm:"-"`
}

type SolicitudPermissions struct {
	CanEdit           bool
	CanApproveReject  bool
	CanRevertApproval bool
	CanMakeDescargo   bool
	CanAssignViatico  bool
	CanPrint          bool
	CanDelete         bool
	CanFinalize       bool
	CanRevertFinalize bool
	CanRegularize     bool
}

type StepView struct {
	Icon         string
	Label        string
	WrapperClass string
	LabelClass   string
}

type StatusCardView struct {
	BorderClass string
	TextClass   string
}

func (s Solicitud) getAuthUser(u ...*Usuario) *Usuario {
	if len(u) > 0 {
		return u[0]
	}
	return s.authUser
}

func (s Solicitud) GetPermissions(u ...*Usuario) SolicitudPermissions {
	return SolicitudPermissions{
		CanEdit:           s.CanEdit(u...),
		CanApproveReject:  s.CanApprove(u...),
		CanRevertApproval: s.CanRevertApproval(u...),
		CanMakeDescargo:   s.CanMakeDescargo(u...),
		CanAssignViatico:  s.CanAssignViatico(u...),
		CanPrint:          s.CanPrint(u...),
		CanDelete:         s.CanDelete(u...),
		CanFinalize:       s.CanFinalize(u...),
		CanRevertFinalize: s.CanRevertFinalize(u...),
		CanRegularize:     s.getAuthUser(u...) != nil && s.getAuthUser(u...).IsAdminOrResponsable(),
	}
}

func (s *Solicitud) HydratePermissions(u ...*Usuario) {
	if len(u) > 0 {
		s.authUser = u[0]
	}
	p := s.GetPermissions()
	s.Permissions = &p

	// Hidratar items y sus pasajes recursivamente
	for i := range s.Items {
		item := &s.Items[i]
		item.HydratePermissions(s.getAuthUser())

		// Hidratar pasajes de cada item
		for j := range item.Pasajes {
			item.Pasajes[j].HydratePermissions(s.getAuthUser())
		}
	}
}

// GetStatusCardClasses retorna las clases de borde y texto para las tarjetas de estado del sistema.
func (s Solicitud) GetStatusCardClasses() (borderClass, textClass string) {
	color := "neutral"
	if s.EstadoSolicitud != nil && s.EstadoSolicitud.Color != "" {
		color = s.EstadoSolicitud.Color
	} else if s.EstadoSolicitudCodigo != nil {
		// Fallback manual si no tiene la relación cargada
		switch *s.EstadoSolicitudCodigo {
		case "SOLICITADO":
			color = "primary"
		case "RECHAZADO":
			color = "danger"
		case "APROBADO":
			color = "success"
		case "PARCIALMENTE_APROBADO":
			color = "violet"
		case "EMITIDO":
			color = "secondary"
		case "FINALIZADO":
			color = "neutral"
		}
	}

	borderClass = fmt.Sprintf("border-%s-500", color)
	textClass = fmt.Sprintf("text-%s-700", color)
	return
}

func (s Solicitud) GetStatusCardData() StatusCardView {
	b, t := s.GetStatusCardClasses()
	return StatusCardView{
		BorderClass: b,
		TextClass:   t,
	}
}

func (s Solicitud) GetStepperData() (map[string]StepView, bool) {
	st := s.GetEstado()

	makeStep := func(active, completed bool, colorBase, icon, label string) StepView {
		sv := StepView{
			Icon:  icon,
			Label: label,
		}
		if active || completed {
			sv.WrapperClass = fmt.Sprintf("bg-%s-500 text-white border-none", colorBase)
			sv.LabelClass = fmt.Sprintf("text-%s-500", colorBase)
		} else {
			sv.WrapperClass = "bg-white border-2 border-neutral-200 text-neutral-400"
			sv.LabelClass = "text-neutral-400"
		}
		return sv
	}

	steps := make(map[string]StepView)
	steps["Solicitado"] = makeStep(true, true, "secondary", "ph ph-file-text text-lg", "Solicitado")

	rejected := st == "RECHAZADO"
	parcial := st == "PARCIALMENTE_APROBADO"

	if rejected {
		steps["Aprobado"] = StepView{
			Icon:         "ph ph-x text-xl font-bold",
			Label:        "Rechazado",
			WrapperClass: "bg-danger-500 text-white border-none",
			LabelClass:   "text-danger-500",
		}
	} else if parcial {
		steps["Aprobado"] = StepView{
			Icon:         "ph ph-check-square-offset text-xl",
			Label:        "Parcial",
			WrapperClass: "bg-violet-500 text-white border-none",
			LabelClass:   "text-violet-500",
		}
	} else {
		isAp := st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
		steps["Aprobado"] = makeStep(isAp, isAp, "success", "ph ph-check text-xl font-bold", "Aprobado")
	}

	isEm := st == "EMITIDO" || st == "FINALIZADO"
	steps["Emitido"] = makeStep(isEm, isEm, "secondary", "ph ph-ticket text-xl", "Emitido")

	isFin := st == "FINALIZADO"
	steps["Finalizado"] = makeStep(isFin, isFin, "neutral", "ph ph-flag-checkered text-xl", "Finalizado")

	return steps, !rejected
}

type StatusFilter struct {
	Codigo string
	Nombre string
	Class  string
}

func (Solicitud) TableName() string {
	return "solicitudes"
}

// Hooks
func (s *Solicitud) BeforeSave(tx *gorm.DB) (err error) {
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) BeforeUpdate(tx *gorm.DB) (err error) {
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s Solicitud) GetEstado() string {
	if s.EstadoSolicitudCodigo == nil {
		return "SOLICITADO"
	}
	return *s.EstadoSolicitudCodigo
}

func (s Solicitud) GetEstadoNombre() string {
	if s.EstadoSolicitud != nil && s.EstadoSolicitud.Nombre != "" {
		return s.EstadoSolicitud.Nombre
	}
	return s.GetEstado()
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

func (s Solicitud) GetAmbitoViajeNombre() string {
	if s.AmbitoViaje != nil {
		return s.AmbitoViaje.Nombre
	}
	return s.AmbitoViajeCodigo
}

func (s *Solicitud) GetItemByID(id string) *SolicitudItem {
	for i := range s.Items {
		if s.Items[i].ID == id {
			return &s.Items[i]
		}
	}
	return nil
}

func (s *Solicitud) UpdateStatusBasedOnItems() {
	if len(s.Items) == 0 {
		return
	}

	// Detección de Itinerario Inteligente: Analizamos qué tramos están HABILITADOS (no pendientes)
	hasItemIda := false
	hasItemVuelta := false
	hasAnyNonPending := false

	for _, item := range s.Items {
		// Solo contamos tramos que NO están pendientes (tienen fecha o han sido habilitados)
		if item.GetEstado() != "PENDIENTE" {
			hasAnyNonPending = true
			switch item.Tipo {
			case TipoSolicitudItemIda:
				hasItemIda = true
			case TipoSolicitudItemVuelta:
				hasItemVuelta = true
			}
		}
	}

	// Si hay tramos habilitados, calculamos basado en ellos.
	if hasAnyNonPending {
		if hasItemIda && hasItemVuelta {
			s.TipoItinerarioCodigo = "IDA_VUELTA"
		} else if hasItemIda {
			s.TipoItinerarioCodigo = "SOLO_IDA"
		} else if hasItemVuelta {
			s.TipoItinerarioCodigo = "SOLO_VUELTA"
		}
	} else {
		// Fallback: Si TODO está pendiente, detectamos por existencia física de los registros
		hasPhysicalIda := false
		hasPhysicalVuelta := false
		for _, item := range s.Items {
			switch item.Tipo {
			case TipoSolicitudItemIda:
				hasPhysicalIda = true
			case TipoSolicitudItemVuelta:
				hasPhysicalVuelta = true
			}
		}

		if hasPhysicalIda && hasPhysicalVuelta {
			s.TipoItinerarioCodigo = "IDA_VUELTA"
		} else if hasPhysicalIda {
			s.TipoItinerarioCodigo = "SOLO_IDA"
		} else if hasPhysicalVuelta {
			s.TipoItinerarioCodigo = "SOLO_VUELTA"
		} else {
			s.TipoItinerarioCodigo = "IDA_VUELTA" // Default estratégico por contexto Senado
		}
	}

	// Estados Individuales para cálculo de Estado Global
	allApproved := true
	allRejected := true
	allFinalized := true
	allEmitidos := true
	hasApproved := false

	for _, item := range s.Items {
		st := item.GetEstado()

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

	// Unificamos criterio: Si TODOS los tramos están en un estado final,
	// la solicitud hereda ese estado global, sin importar si es solo IDA o solo VUELTA.
	newState := "SOLICITADO"

	if allFinalized {
		newState = "FINALIZADO"
	} else if allEmitidos {
		newState = "EMITIDO"
	} else if allApproved {
		newState = "APROBADO"
	} else if hasApproved {
		newState = "PARCIALMENTE_APROBADO"
	} else if allRejected {
		newState = "RECHAZADO"
	} else {
		newState = "SOLICITADO"
	}

	s.EstadoSolicitudCodigo = &newState
}

// Actions

func (s *Solicitud) CanApprove(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	estado := s.GetEstado()
	return estado == "SOLICITADO" || estado == "PARCIALMENTE_APROBADO"
}

func (s *Solicitud) CanReject(u ...*Usuario) bool {
	return s.CanApprove(u...) // Misma lógica: se rechaza si se puede aprobar
}

func (s *Solicitud) CanRevertApproval(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	st := s.GetEstado()
	canRevertState := st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO"
	return canRevertState && !s.HasEmittedPasaje()
}

func (s *Solicitud) Approve() error {
	// Usamos el nuevo método de permiso (aunque en procesos internos pasemos nil al usuario si no aplica check de rol)
	st := s.GetEstado()
	if st != "SOLICITADO" && st != "PARCIALMENTE_APROBADO" {
		return errors.New("la solicitud no está en un estado que permita aprobación")
	}

	hasIda := s.GetItemIda() != nil
	hasVuelta := s.GetItemVuelta() != nil
	// Si tiene ambos tramos, aprobamos ambos por defecto.
	// Si solo hay uno, aprobamos los que están en SOLICITADO y no son marcados como PENDIENTE (sin fecha).
	approveAll := hasIda && hasVuelta

	for i := range s.Items {
		item := &s.Items[i]
		shouldApprove := false

		if approveAll {
			shouldApprove = true
		} else {
			// Approve only if it has a confirmed date (i.e., not PENDING placeholder)
			if item.GetEstado() != "PENDIENTE" {
				shouldApprove = true
			}
		}

		if shouldApprove && item.GetEstado() == "SOLICITADO" {
			item.Approve()
		}
	}
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) Reject() error {
	if s.GetEstado() != "SOLICITADO" && s.GetEstado() != "PARCIALMENTE_APROBADO" {
		return errors.New("la solicitud no está en un estado que permita rechazo")
	}
	for i := range s.Items {
		s.Items[i].Reject()
	}
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) Finalize() error {
	if s.GetEstado() != "EMITIDO" {
		return errors.New("la solicitud debe estar en estado EMITIDO para ser finalizada")
	}
	for i := range s.Items {
		s.Items[i].Finalize()
	}
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) RevertApproval() error {
	// Revertir solo si no tiene pasajes emitidos (esto lo valida el servicio usualmente, pero pongamos base aquí)
	if s.GetEstado() == "FINALIZADO" {
		return errors.New("no se puede revertir una solicitud ya finalizada")
	}

	for i := range s.Items {
		item := &s.Items[i]
		if item.GetEstado() == "APROBADO" {
			item.RevertApproval()
		}
	}
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) RevertFinalize() error {
	if s.GetEstado() != "FINALIZADO" {
		return errors.New("la solicitud no está en estado FINALIZADO")
	}
	for i := range s.Items {
		item := &s.Items[i]
		item.RevertFinalize()
	}
	s.UpdateStatusBasedOnItems()
	return nil
}

func (s *Solicitud) ApproveItem(itemID string) bool {
	for i := range s.Items {
		if s.Items[i].ID == itemID {
			s.Items[i].Approve()
			s.UpdateStatusBasedOnItems()
			return true
		}
	}
	return false
}

func (s *Solicitud) RejectItem(itemID string) bool {
	for i := range s.Items {
		if s.Items[i].ID == itemID {
			s.Items[i].Reject()
			s.UpdateStatusBasedOnItems()
			return true
		}
	}
	return false
}

func (s *Solicitud) RevertApprovalItem(itemID string) bool {
	for i := range s.Items {
		if s.Items[i].ID == itemID {
			s.Items[i].RevertApproval()
			s.UpdateStatusBasedOnItems()
			return true
		}
	}
	return false
}

func (s *Solicitud) AreAllItemsInactive() bool {
	if len(s.Items) == 0 {
		return true
	}
	for _, it := range s.Items {
		st := it.GetEstado()
		// Consideramos inactivos: RECHAZADO, CANCELADO, PENDIENTE
		if st != "RECHAZADO" && st != "CANCELADO" && st != "PENDIENTE" {
			return false
		}
	}
	return true
}

func (s *Solicitud) GetEditableItemIDs() []string {
	var ids []string
	for _, it := range s.Items {
		if it.CanEdit() {
			ids = append(ids, it.ID)
		}
	}
	return ids
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
	// Look for IDA leg first
	for i := range s.Items {
		if s.Items[i].Tipo == TipoSolicitudItemIda {
			return s.Items[i].Origen
		}
	}
	// Fallback to first item's origin
	if len(s.Items) > 0 {
		return s.Items[0].Origen
	}
	return nil
}

func (s Solicitud) GetDestino() *Destino {
	// En giras u oficiales, el destino es el destino del ÚLTIMO tramo de IDA.
	var lastIda *SolicitudItem
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "IDA" {
			lastIda = &s.Items[i]
		}
	}

	if lastIda != nil {
		return lastIda.Destino
	}

	// Fallback a VUELTA solo si no hay IDA (caso rarísimo)
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "VUELTA" {
			return s.Items[i].Destino
		}
	}

	// Fallback final
	if len(s.Items) > 0 {
		return s.Items[len(s.Items)-1].Destino
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

func (s Solicitud) GetRutaDetallada() string {
	origen := s.GetOrigen()
	if origen == nil {
		return "-"
	}
	destino := s.GetDestino()
	if destino == nil {
		return "-"
	}
	return origen.GetLabel() + " - " + destino.GetLabel()
}

func (s Solicitud) GetOrigenLabel() string {
	obj := s.GetOrigen()
	if obj == nil {
		return "-"
	}
	return obj.GetLabel()
}

func (s Solicitud) GetDestinoLabel() string {
	obj := s.GetDestino()
	if obj == nil {
		return "-"
	}
	return obj.GetLabel()
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
	// Para el regreso, buscamos el ÚLTIMO ítem de tipo VUELTA (el retorno final)
	var lastVuelta *SolicitudItem
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "VUELTA" {
			lastVuelta = &s.Items[i]
		}
	}
	return lastVuelta
}

func (s Solicitud) GetAllItemsIda() []*SolicitudItem {
	var items []*SolicitudItem
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "IDA" {
			items = append(items, &s.Items[i])
		}
	}
	return items
}

func (s Solicitud) GetAllItemsVuelta() []*SolicitudItem {
	var items []*SolicitudItem
	for i := range s.Items {
		if strings.ToUpper(string(s.Items[i].Tipo)) == "VUELTA" {
			items = append(items, &s.Items[i])
		}
	}
	return items
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

func (s Solicitud) GetDiasRestantesDescargo() int {
	maxDate := s.GetMaxFechaVueloEmitida()
	if maxDate == nil {
		return 999
	}

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

func (s Solicitud) CanMarkUsado(u *Usuario) bool {
	if u == nil {
		return false
	}
	if u.IsAdminOrResponsable() {
		return true
	}
	if u.ID == s.UsuarioID {
		return true
	}
	if s.Usuario.EncargadoID != nil && *s.Usuario.EncargadoID == u.ID {
		return true
	}
	if s.CreatedBy != nil && *s.CreatedBy == u.ID {
		return true
	}
	return false
}

func (s Solicitud) CanEdit(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil {
		return false
	}
	// El Administrador o Responsable siempre tiene poder de edición administrativa
	if user.IsAdminOrResponsable() {
		return s.CanView(user)
	}

	// Para usuarios normales, solo si está en estados editables
	estado := s.GetEstado()
	isEditableState := (estado == "SOLICITADO" || estado == "RECHAZADO" || estado == "PARCIALMENTE_APROBADO")
	if !isEditableState {
		return false
	}

	return s.CanView(user)
}

func (s Solicitud) IsDeletableState() bool {
	return s.GetEstado() == "SOLICITADO"
}

func (s Solicitud) CanDelete(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !s.IsDeletableState() {
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

func (s Solicitud) CanAssignPasaje(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	st := s.GetEstado()
	// Solo tiene sentido asignar pasajes en flujos aprobados o activos
	return st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
}

func (s Solicitud) CanAssignViatico(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	st := s.GetEstado()
	return st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
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

func (s Solicitud) CanMakeDescargo(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	return s.HasEmittedPasaje() && s.CanView(user)
}

func (s Solicitud) CanPrint(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	// Solo se imprime si está aprobada o emitida
	st := s.GetEstado()
	return (st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO" || st == "PARCIALMENTE_APROBADO") && s.CanView(user)
}

func (s Solicitud) CanFinalize(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	return s.GetEstado() == "EMITIDO"
}

func (s Solicitud) CanRevertFinalize(u ...*Usuario) bool {
	user := s.getAuthUser(u...)
	if user == nil || !user.IsAdminOrResponsable() {
		return false
	}
	return s.GetEstado() == "FINALIZADO"
}

func (s Solicitud) GetTipoItinerarioNombre() string {
	if s.TipoItinerario != nil && s.TipoItinerario.Nombre != "" {
		return s.TipoItinerario.Nombre
	}
	return s.TipoItinerarioCodigo
}

func (s Solicitud) IsSoloIda() bool {
	return s.TipoItinerarioCodigo == "SOLO_IDA"
}

func (s Solicitud) IsSoloVuelta() bool {
	return s.TipoItinerarioCodigo == "SOLO_VUELTA"
}

func (s Solicitud) IsIdaVuelta() bool {
	return s.TipoItinerarioCodigo == "IDA_VUELTA"
}

func (s Solicitud) IsSolicitado() bool {
	return s.GetEstado() == "SOLICITADO"
}

func (s Solicitud) IsAprobado() bool {
	return s.GetEstado() == "APROBADO"
}

func (s Solicitud) IsParcialmenteAprobado() bool {
	return s.GetEstado() == "PARCIALMENTE_APROBADO"
}

func (s Solicitud) IsRechazado() bool {
	return s.GetEstado() == "RECHAZADO"
}

func (s Solicitud) IsEmitido() bool {
	return s.GetEstado() == "EMITIDO"
}

func (s Solicitud) IsFinalizado() bool {
	return s.GetEstado() == "FINALIZADO"
}

// --- Display Helpers ---

func (s Solicitud) IsDerecho() bool {
	return s.GetConceptoCodigo() == "DERECHO"
}

func (s Solicitud) IsOficial() bool {
	return s.GetConceptoCodigo() == "OFICIAL"
}

func (s Solicitud) GetStatusBadgeClass() string {
	if s.EstadoSolicitud != nil && s.EstadoSolicitud.Color != "" {
		color := s.EstadoSolicitud.Color
		// Casos especiales (neutral black, success bold, etc)
		if color == "neutral-800" || color == "black" {
			return "bg-neutral-800 text-white"
		}
		if s.GetEstado() == "APROBADO" || s.GetEstado() == "EMITIDO" {
			return fmt.Sprintf("bg-%s-100 text-%s-700 font-bold", color, color)
		}
		return fmt.Sprintf("bg-%s-100 text-%s-700", color, color)
	}

	// Fallback por defecto si no hay relación cargada
	return "bg-neutral-100 text-neutral-600"
}

func GetAvailableStatuses() []StatusFilter {
	// Nota: Estos filtros podrían moverse a un repositorio para ser cargados desde la DB
	return []StatusFilter{
		{Codigo: "SOLICITADO", Nombre: "Solicitados", Class: "bg-primary"},
		{Codigo: "APROBADO", Nombre: "Aprobados", Class: "bg-success-600"},
		{Codigo: "EMITIDO", Nombre: "Emitidos", Class: "bg-secondary"},
		{Codigo: "RECHAZADO", Nombre: "Rechazados", Class: "bg-danger-600"},
		{Codigo: "FINALIZADO", Nombre: "Finalizados", Class: "bg-neutral-600"},
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

	// 1. Validar Itinerario (Billete, Pase, Archivo)
	hasItinerary := false
	for _, it := range s.Descargo.Tramos {
		if !it.EsDevolucion {
			hasItinerary = true
			if it.Billete == "" || it.NumeroPaseAbordo == "" || it.ArchivoPaseAbordo == "" {
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
	missingBilletes := false
	missingPases := false
	missingArchivos := false

	for _, it := range s.Descargo.Tramos {
		if !it.EsDevolucion {
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

	if !hasItinerary && !s.Descargo.allDevolucion() {
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
	if s.AerolineaID != old.AerolineaID {
		changes["aerolinea_id"] = s.AerolineaID
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
