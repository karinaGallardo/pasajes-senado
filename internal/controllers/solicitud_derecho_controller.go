package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudDerechoController struct {
	solicitudService      *services.SolicitudService
	destinoService        *services.DestinoService
	conceptoService       *services.ConceptoService
	tipoSolicitudService  *services.TipoSolicitudService
	ambitoService         *services.AmbitoService
	cupoService           *services.CupoService
	userService           *services.UsuarioService
	peopleService         *services.PeopleService
	reportService         *services.ReportService
	aerolineaService      *services.AerolineaService
	agenciaService        *services.AgenciaService
	tipoItinerarioService *services.TipoItinerarioService
	rutaService           *services.RutaService
}

func NewSolicitudDerechoController() *SolicitudDerechoController {
	return &SolicitudDerechoController{
		solicitudService:      services.NewSolicitudService(),
		destinoService:        services.NewDestinoService(),
		conceptoService:       services.NewConceptoService(),
		tipoSolicitudService:  services.NewTipoSolicitudService(),
		ambitoService:         services.NewAmbitoService(),
		cupoService:           services.NewCupoService(),
		userService:           services.NewUsuarioService(),
		peopleService:         services.NewPeopleService(),
		reportService:         services.NewReportService(),
		aerolineaService:      services.NewAerolineaService(),
		agenciaService:        services.NewAgenciaService(),
		tipoItinerarioService: services.NewTipoItinerarioService(),
		rutaService:           services.NewRutaService(),
	}
}

func (ctrl *SolicitudDerechoController) Create(c *gin.Context) {

	authUser := appcontext.AuthUser(c)

	itemID := c.Param("item_id")
	itinerarioCode := c.Param("itinerario_code")

	item, err := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	itinerario, err := ctrl.tipoItinerarioService.GetByCodigo(c.Request.Context(), itinerarioCode)
	if err != nil {
		c.String(http.StatusBadRequest, "Itinerario inválido")
		return
	}

	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), item.SenAsignadoID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Usuario titular del derecho no encontrado")
		return
	}

	canCreate := false
	if authUser.ID == targetUser.ID {
		canCreate = true
	} else if authUser.IsAdminOrResponsable() {
		canCreate = true
	} else if targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID {
		canCreate = true
	}

	if !canCreate {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta solicitud")
		return
	}

	var alertaOrigen string
	if targetUser.GetOrigenIATA() == "" {
		alertaOrigen = "Este usuario no tiene configurado su LUGAR DE ORIGEN en el perfil. El sistema no podrá calcular rutas automáticamente."
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	tipoSolicitud, ambitoNac, _ := ctrl.tipoSolicitudService.GetByCodigoAndAmbito(c.Request.Context(), "USO_CUPO", "NACIONAL")

	weekDays := ctrl.cupoService.GetCupoDerechoItemWeekDays(item)

	origenIATA := targetUser.GetOrigenIATA()

	var origen, destino *models.Destino

	userLoc, err := ctrl.destinoService.GetByIATA(c.Request.Context(), origenIATA)
	if err != nil || userLoc == nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/usuarios/%s/editar", targetUser.ID))
		return
	}

	lpbLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), "LPB")

	if itinerario.Codigo == "SOLO_IDA" || itinerario.Codigo == "IDA_VUELTA" {
		origen = userLoc
		destino = lpbLoc
	} else {
		origen = lpbLoc
		destino = userLoc

		existingSolicitudes, _ := ctrl.solicitudService.GetByCupoDerechoItemID(c.Request.Context(), itemID)
		var fechaIda *time.Time
		for _, sol := range existingSolicitudes {
			if sol.TipoItinerario.Codigo == "SOLO_IDA" {
				stateCode := sol.GetEstadoCodigo()
				if stateCode != "RECHAZADO" && stateCode != "ELIMINADO" {
					for _, item := range sol.Items {
						if item.Fecha != nil {
							fechaIda = item.Fecha
							break
						}
					}
				}
			}
			if fechaIda != nil {
				break
			}
		}

		if fechaIda != nil {
			var filteredDays []map[string]string
			fechaIdaStr := fechaIda.Format("2006-01-02")
			for _, d := range weekDays {
				if d["date"] > fechaIdaStr {
					filteredDays = append(filteredDays, d)
				}
			}
			weekDays = filteredDays
		}
	}

	data := gin.H{
		"Title":        "Pasaje por Derecho - " + targetUser.GetNombreCompleto(),
		"TargetUser":   targetUser,
		"Aerolineas":   aerolineas,
		"AlertaOrigen": alertaOrigen,
		"Item":         item,
		"WeekDays":     weekDays,

		"Concepto":      tipoSolicitud.ConceptoViaje,
		"TipoSolicitud": tipoSolicitud,
		"Ambito":        ambitoNac,

		"Itinerario": itinerario,
		"Origen":     origen,
		"Destino":    destino,
	}
	utils.Render(c, "solicitud/derecho/create", data)
}

func (ctrl *SolicitudDerechoController) GetCreateModal(c *gin.Context) {
	itemID := c.Param("item_id")
	itinerarioCode := c.Param("itinerario_code")

	item, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	itinerario, _ := ctrl.tipoItinerarioService.GetByCodigo(c.Request.Context(), itinerarioCode)
	targetUser, _ := ctrl.userService.GetByID(c.Request.Context(), item.SenAsignadoID)

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	tipoSolicitud, ambitoNac, _ := ctrl.tipoSolicitudService.GetByCodigoAndAmbito(c.Request.Context(), "USO_CUPO", "NACIONAL")

	// Calculate Min/Max Dates for Calendar
	// Calculate Min/Max Dates for Calendar (Extended Range: +/- 2 weeks)
	var minDateIda string
	if item.FechaDesde != nil {
		minDateIda = item.FechaDesde.AddDate(0, 0, -14).Format("2006-01-02")
	} else {
		minDateIda = time.Now().AddDate(0, 0, -14).Format("2006-01-02")
	}

	var maxDateIda string
	if item.FechaHasta != nil {
		maxDateIda = item.FechaHasta.AddDate(0, 0, 14).Format("2006-01-02")
	}

	minDateVuelta := minDateIda
	maxDateVuelta := maxDateIda

	var origen, destino *models.Destino
	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), targetUser.GetOrigenIATA())
	lpbLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), "LPB")

	if itinerario.Codigo == "SOLO_IDA" || itinerario.Codigo == "IDA_VUELTA" {
		origen = userLoc
		destino = lpbLoc
	} else {
		origen = lpbLoc
		destino = userLoc
	}

	referer := c.Request.Referer()
	if referer == "" {
		referer = "/cupos/derecho/" + targetUser.ID
	}

	utils.Render(c, "solicitud/derecho/modal_form", gin.H{
		"TargetUser":        targetUser,
		"Aerolineas":        aerolineas,
		"Item":              item,
		"Concepto":          tipoSolicitud.ConceptoViaje,
		"TipoSolicitud":     tipoSolicitud,
		"Ambito":            ambitoNac,
		"Itinerario":        itinerario,
		"Origen":            origen,
		"Destino":           destino,
		"IsEdit":            false,
		"ReturnURL":         referer,
		"MinDateIda":        minDateIda,
		"MaxDateIda":        maxDateIda,
		"MinDateVuelta":     minDateVuelta,
		"MaxDateVuelta":     maxDateVuelta,
		"DefaultDateIda":    time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		"DefaultDateVuelta": time.Now().AddDate(0, 0, 4).Format("2006-01-02"),
	})
}

func (ctrl *SolicitudDerechoController) Store(c *gin.Context) {
	var req dtos.CreateSolicitudRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	if req.CupoDerechoItemID == "" {
		c.String(http.StatusBadRequest, "ID de Registro de Derecho requerido para solicitud")
		return
	}

	authUser := appcontext.AuthUser(c)

	solicitud, err := ctrl.solicitudService.CreateDerecho(c.Request.Context(), req, authUser)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creando solicitud: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud creada correctamente")
	targetURL := fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}
	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *SolicitudDerechoController) Edit(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	// Sort items (IDA first, then VUELTA)
	sort.Slice(solicitud.Items, func(i, j int) bool {
		ti := string(solicitud.Items[i].Tipo)
		tj := string(solicitud.Items[j].Tipo)

		if ti == "IDA" && tj == "VUELTA" {
			return true
		}
		if ti == "VUELTA" && tj == "IDA" {
			return false
		}

		// Chronological fallback if same type or unknown
		if solicitud.Items[i].Fecha != nil && solicitud.Items[j].Fecha != nil {
			return solicitud.Items[i].Fecha.Before(*solicitud.Items[j].Fecha)
		}
		return false
	})

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No se puede editar una solicitud que no está en estado SOLICITADO")
		return
	}

	authUser := appcontext.AuthUser(c)

	if !authUser.CanEditSolicitud(*solicitud) {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	if solicitud.CupoDerechoItemID == nil {
		c.String(http.StatusBadRequest, "Esta solicitud no corresponde a un pasaje por derecho")
		return
	}

	item, err := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), *solicitud.CupoDerechoItemID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Derecho asociado no encontrado")
		return
	}

	TiposItinerario, _ := ctrl.tipoItinerarioService.GetAll(c.Request.Context())
	var ItinerarioIdaID string
	var ItinerarioVueltaID string
	var ActiveTab string

	for _, t := range TiposItinerario {
		if t.Codigo == "SOLO_IDA" {
			ItinerarioIdaID = t.Codigo
		}
		if t.Codigo == "SOLO_VUELTA" {
			ItinerarioVueltaID = t.Codigo
		}
	}

	if solicitud.TipoItinerario != nil {
		ActiveTab = solicitud.TipoItinerario.Codigo
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	origenIATA := solicitud.Usuario.GetOrigenIATA()
	userLoc, err := ctrl.destinoService.GetByIATA(c.Request.Context(), origenIATA)
	if err != nil || userLoc == nil {
		c.String(http.StatusInternalServerError, "Usuario sin origen configurado")
		return
	}
	lpbLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), "LPB")

	var origen, destino *models.Destino
	if ActiveTab == "SOLO_IDA" || ActiveTab == "IDA_VUELTA" {
		origen = userLoc
		destino = lpbLoc
	} else {
		origen = lpbLoc
		destino = userLoc
	}

	weekDays := []gin.H{}
	if item.FechaDesde != nil {
		start := *item.FechaDesde
		names := []string{"Dom", "Lun", "Mar", "Mie", "Jue", "Vie", "Sab"}
		for i := 0; i < 7; i++ {
			d := start.AddDate(0, 0, i)
			esName := names[d.Weekday()]
			weekDays = append(weekDays, gin.H{
				"date":   d.Format("2006-01-02"),
				"name":   esName,
				"dayNum": d.Format("02"),
			})
		}
	}

	data := gin.H{
		"Aerolineas":         aerolineas,
		"TargetUser":         &solicitud.Usuario,
		"Itinerarios":        TiposItinerario,
		"ItinerarioIdaID":    ItinerarioIdaID,
		"ItinerarioVueltaID": ItinerarioVueltaID,
		"Item":               item,
		"ItemID":             item.ID,
		"ActiveTab":          ActiveTab,
		"Solicitud":          solicitud,
		"IsEdit":             true,
		"WeekDays":           weekDays,
		"Origen":             origen,
		"Destino":            destino,
		"Concepto":           solicitud.TipoSolicitud.ConceptoViaje,
		"TipoSolicitud":      solicitud.TipoSolicitud,
		"Ambito":             solicitud.AmbitoViaje,
		"Itinerario":         solicitud.TipoItinerario,
	}
	utils.Render(c, "solicitud/derecho/edit", data)
}

func (ctrl *SolicitudDerechoController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	item, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), *solicitud.CupoDerechoItemID)
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	origenIATA := solicitud.Usuario.GetOrigenIATA()
	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), origenIATA)
	lpbLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), "LPB")

	var origen, destino *models.Destino
	if solicitud.TipoItinerario.Codigo == "SOLO_IDA" || solicitud.TipoItinerario.Codigo == "IDA_VUELTA" {
		origen = userLoc
		destino = lpbLoc
	} else {
		origen = lpbLoc
		destino = userLoc
	}

	// Calculate Return Days (Current Week + Next Week)
	// Calculate Min/Max Dates for Calendar (Edit Mode)
	// Calculate Min/Max Dates for Calendar (Extended Range: +/- 2 weeks)
	var minDateIda string
	if item.FechaDesde != nil {
		minDateIda = item.FechaDesde.AddDate(0, 0, -14).Format("2006-01-02")
	} else {
		minDateIda = time.Now().AddDate(0, 0, -14).Format("2006-01-02")
	}

	var maxDateIda string
	if item.FechaHasta != nil {
		maxDateIda = item.FechaHasta.AddDate(0, 0, 14).Format("2006-01-02")
	}

	minDateVuelta := minDateIda
	maxDateVuelta := maxDateIda

	referer := c.Request.Referer()
	if referer == "" {
		referer = "/cupos/derecho/" + solicitud.Usuario.ID
	}

	utils.Render(c, "solicitud/derecho/modal_form", gin.H{
		"Aerolineas":    aerolineas,
		"TargetUser":    &solicitud.Usuario,
		"Item":          item,
		"Solicitud":     solicitud,
		"IsEdit":        true,
		"Concepto":      solicitud.TipoSolicitud.ConceptoViaje,
		"TipoSolicitud": solicitud.TipoSolicitud,
		"Ambito":        solicitud.AmbitoViaje,
		"Itinerario":    solicitud.TipoItinerario,
		"ReturnURL":     referer,
		"Origen":        origen,
		"Destino":       destino,
		"MinDateIda":    minDateIda,
		"MaxDateIda":    maxDateIda,
		"MinDateVuelta": minDateVuelta,
		"MaxDateVuelta": maxDateVuelta,
	})
}

func (ctrl *SolicitudDerechoController) Update(c *gin.Context) {
	id := c.Param("id")
	var req dtos.UpdateSolicitudRequest

	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	} else {
		c.String(http.StatusBadRequest, "Formato fecha salida inválido")
		return
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" {
		if t, err := time.Parse(layout, req.FechaVuelta); err == nil {
			fechaVuelta = &t
		} else {
			c.String(http.StatusBadRequest, "Formato fecha retorno inválido")
			return
		}
	}

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	currentStatus := "SOLICITADO"
	if solicitud.EstadoSolicitudCodigo != nil {
		currentStatus = *solicitud.EstadoSolicitudCodigo
	}

	if currentStatus != "SOLICITADO" && currentStatus != "APROBADO" && currentStatus != "PARCIALMENTE_APROBADO" && currentStatus != "RECHAZADO" {
		c.String(http.StatusForbidden, "No editable. El estado actual ("+currentStatus+") no permite modificaciones.")
		return
	}

	if req.TipoItinerarioCodigo != "" {
		solicitud.TipoItinerarioCodigo = req.TipoItinerarioCodigo
	}

	if currentStatus == "SOLICITADO" || currentStatus == "APROBADO" || currentStatus == "PARCIALMENTE_APROBADO" || currentStatus == "RECHAZADO" {
		solicitud.TipoSolicitudCodigo = req.TipoSolicitudCodigo
		solicitud.AmbitoViajeCodigo = req.AmbitoViajeCodigo
		solicitud.AerolineaSugerida = req.AerolineaSugerida

		orig, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), req.OrigenIATA)
		dest, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), req.DestinoIATA)

		for i := range solicitud.Items {
			it := &solicitud.Items[i]
			st := it.GetEstado()

			switch it.Tipo {
			case models.TipoSolicitudItemIda:
				it.Fecha = fechaIda
				it.Hora = fechaIda.Format("15:04")
				// Al editar la ida, si estaba rechazado vuelve a solicitado
				if st == "RECHAZADO" {
					it.EstadoCodigo = utils.Ptr("SOLICITADO")
				}
				if orig != nil {
					it.OrigenIATA = orig.IATA
					it.Origen = orig
				}
				if dest != nil {
					it.DestinoIATA = dest.IATA
					it.Destino = dest
				}
			case models.TipoSolicitudItemVuelta:
				it.Fecha = fechaVuelta
				if fechaVuelta != nil {
					it.Hora = fechaVuelta.Format("15:04")
					// Si estaba pendiente (sin fecha) o rechazado y ahora tiene fecha, vuelve a SOLICITADO
					if st == "PENDIENTE" || st == "RECHAZADO" {
						it.EstadoCodigo = utils.Ptr("SOLICITADO")
					}
				} else {
					// Si no tiene fecha (vuelta por confirmar) y estaba SOLICITADO o RECHAZADO, pasa a PENDIENTE
					if st == "SOLICITADO" || st == "RECHAZADO" {
						it.EstadoCodigo = utils.Ptr("PENDIENTE")
					}
				}
				// Para la vuelta, el destino de llegada es el origen del viaje (regresa a casa)
				if orig != nil {
					it.DestinoIATA = orig.IATA
					it.Destino = orig
				}
				if dest != nil {
					it.OrigenIATA = dest.IATA
					it.Origen = dest
				}
			}
		}
	}
	solicitud.Motivo = req.Motivo

	// If it was APPROVED, PARCIAL or REJECTED, revert to SOLICITADO to trigger re-approval of adjustments
	if currentStatus == "APROBADO" || currentStatus == "PARCIALMENTE_APROBADO" || currentStatus == "RECHAZADO" {
		solicitud.EstadoSolicitudCodigo = utils.Ptr("SOLICITADO")
	}

	if err := ctrl.solicitudService.Update(c.Request.Context(), solicitud); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud actualizada correctamente")

	targetURL := fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}
	c.Redirect(http.StatusFound, targetURL)
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

type SolicitudPermissions struct {
	CanEdit           bool
	CanApproveReject  bool
	CanRevertApproval bool
	CanMakeDescargo   bool
	CanAssignPasaje   bool
	CanAssignViatico  bool
	IsAdminOrResp     bool
}

type PasajePermissions struct {
	CanEdit         bool
	CanMarkUsado    bool
	CanValidateUso  bool
	CanDevolver     bool
	CanAnular       bool
	CanEmitir       bool
	ShowActionsMenu bool
}

type PasajeView struct {
	models.Pasaje
	Perms            PasajePermissions
	StatusColorClass string // e.g. "bg-success-100 text-success-800"
}

func (ctrl *SolicitudDerechoController) Show(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	authUser := appcontext.AuthUser(c)

	// --- 0. Sort Items (IDA first, then VUELTA) ---
	sort.Slice(solicitud.Items, func(i, j int) bool {
		ti := string(solicitud.Items[i].Tipo)
		tj := string(solicitud.Items[j].Tipo)

		if ti == "IDA" && tj == "VUELTA" {
			return true
		}
		if ti == "VUELTA" && tj == "IDA" {
			return false
		}

		if solicitud.Items[i].Fecha != nil && solicitud.Items[j].Fecha != nil {
			return solicitud.Items[i].Fecha.Before(*solicitud.Items[j].Fecha)
		}
		return false
	})

	st := "SOLICITADO"
	if solicitud.EstadoSolicitudCodigo != nil {
		st = *solicitud.EstadoSolicitudCodigo
	}

	canView := false
	if authUser.IsAdminOrResponsable() || solicitud.UsuarioID == authUser.ID || (solicitud.CreatedBy != nil && *solicitud.CreatedBy == authUser.ID) {
		canView = true
	} else if solicitud.Usuario.EncargadoID != nil && *solicitud.Usuario.EncargadoID == authUser.ID {
		canView = true
	}

	if !canView {
		c.String(http.StatusForbidden, "No tiene permiso para ver esta solicitud")
		return
	}

	hasEmitted := false
	for _, item := range solicitud.Items {
		for _, p := range item.Pasajes {
			if p.EstadoPasajeCodigo != nil && *p.EstadoPasajeCodigo == "EMITIDO" {
				hasEmitted = true
				break
			}
		}
		if hasEmitted {
			break
		}
	}

	perms := SolicitudPermissions{
		CanEdit:           authUser.CanEditSolicitud(*solicitud),
		CanApproveReject:  authUser.CanApproveReject(),
		CanRevertApproval: authUser.IsAdminOrResponsable() && (st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO") && !hasEmitted,
		CanAssignPasaje:   authUser.IsAdminOrResponsable(),
		CanMakeDescargo:   hasEmitted,
		IsAdminOrResp:     authUser.IsAdminOrResponsable(),
	}

	approvalLabel := "Acciones"

	// --- 2. Stepper Logic ---
	// Helper to create step style
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

	// Step 1: Solicitado (Always active/completed) (Blue -> Primary)
	steps["Solicitado"] = makeStep(true, true, "primary", "ph ph-file-text text-lg", "Solicitado")

	// Step 2: Aprobado / Parcial / Rechazado
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
		// Aprobado is active if state is APROBADO, EMITIDO, or FINALIZADO
		isAp := st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
		steps["Aprobado"] = makeStep(isAp, isAp, "success", "ph ph-check text-xl font-bold", "Aprobado")
	}

	// Step 3: Emitido (Teal -> Secondary)
	isEm := st == "EMITIDO" || st == "FINALIZADO"
	steps["Emitido"] = makeStep(isEm, isEm, "secondary", "ph ph-ticket text-xl", "Emitido")

	// Step 4: Finalizado (Gray -> Neutral)
	isFin := st == "FINALIZADO"
	steps["Finalizado"] = makeStep(isFin, isFin, "neutral", "ph ph-flag-checkered text-xl", "Finalizado")

	showNextSteps := !rejected

	// --- 3. Status Card Logic ---
	statusCard := StatusCardView{}
	switch st {
	case "SOLICITADO":
		statusCard.BorderClass = "border-primary-500"
		statusCard.TextClass = "text-primary-700"
	case "RECHAZADO":
		statusCard.BorderClass = "border-danger-500"
		statusCard.TextClass = "text-danger-700"
	case "APROBADO":
		statusCard.BorderClass = "border-success-500"
		statusCard.TextClass = "text-success-700"
	case "PARCIALMENTE_APROBADO":
		statusCard.BorderClass = "border-violet-500"
		statusCard.TextClass = "text-violet-700"
	case "EMITIDO":
		statusCard.BorderClass = "border-secondary-500"
		statusCard.TextClass = "text-secondary-700"
	case "FINALIZADO":
		statusCard.BorderClass = "border-neutral-500"
		statusCard.TextClass = "text-neutral-700"
	default:
		statusCard.BorderClass = "border-neutral-200"
		statusCard.TextClass = "text-neutral-900"
	}

	// --- 4. Pasajes Views (con permisos pre-calculados) ---
	var pasajesViews []PasajeView
	for _, item := range solicitud.Items {
		for i := range item.Pasajes {
			p := &item.Pasajes[i]
			pCode := ""
			if p.EstadoPasajeCodigo != nil {
				pCode = *p.EstadoPasajeCodigo
			}

			pv := PasajeView{Pasaje: *p}

			// Status Color logic
			if p.EstadoPasaje != nil {
				pv.StatusColorClass = fmt.Sprintf("bg-%s-100 text-%s-800", p.EstadoPasaje.Color, p.EstadoPasaje.Color)
			} else {
				// Fallback logic
				switch pCode {
				case "REGISTRADO":
					pv.StatusColorClass = "bg-secondary-100 text-secondary-800"
				case "EMITIDO":
					pv.StatusColorClass = "bg-success-100 text-success-800"
				case "USADO":
					pv.StatusColorClass = "bg-primary-100 text-primary-800"
				case "ANULADO":
					pv.StatusColorClass = "bg-neutral-100 text-neutral-800"
				default:
					pv.StatusColorClass = "bg-neutral-100 text-neutral-800"
				}
			}

			// Permissions logic for this pasaje
			pPerms := PasajePermissions{}

			if item.GetEstado() != "REPROGRAMADO" {
				// Individual Actions
				canEmitir := authUser.IsAdminOrResponsable() && pCode == "REGISTRADO"
				pPerms.CanEdit = authUser.IsAdminOrResponsable() && pCode == "REGISTRADO"
				pPerms.CanMarkUsado = authUser.CanMarkUsado(*solicitud) && pCode == "EMITIDO"
				pPerms.CanValidateUso = false // Removed state
				pPerms.CanDevolver = authUser.IsAdminOrResponsable() && pCode == "EMITIDO"
				pPerms.CanAnular = authUser.IsAdminOrResponsable() && (pCode == "REGISTRADO" || pCode == "EMITIDO")
				pPerms.CanEmitir = canEmitir // New permission

				// Actions Menu Visibility
				pPerms.ShowActionsMenu = pPerms.CanEdit || pPerms.CanMarkUsado || pPerms.CanDevolver || pPerms.CanAnular || pPerms.CanEmitir
			} else {
				pPerms.ShowActionsMenu = false
			}

			pv.Perms = pPerms
			pasajesViews = append(pasajesViews, pv)
		}
	}
	// Dependencies
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	userIDsMap := make(map[string]bool)
	if solicitud.CreatedBy != nil {
		userIDsMap[*solicitud.CreatedBy] = true
	}
	if solicitud.UpdatedBy != nil {
		userIDsMap[*solicitud.UpdatedBy] = true
	}
	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}
	usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
	usuariosMap := make(map[string]*models.Usuario)
	for i := range usuarios {
		usuariosMap[usuarios[i].ID] = &usuarios[i]
	}

	utils.Render(c, "solicitud/derecho/show", gin.H{
		"Title":     "Detalle Solicitud (Derecho) #" + id,
		"Solicitud": solicitud,
		"Usuarios":  usuariosMap,

		// New View Data
		"Perms":         perms,
		"Steps":         steps,
		"ShowNextSteps": showNextSteps,
		"StatusCard":    statusCard,
		"PasajesView":   pasajesViews,
		"ApprovalLabel": approvalLabel,

		"Aerolineas": aerolineas,
	})
}

func (ctrl *SolicitudDerechoController) Approve(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.Approve(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error al aprobar la solicitud: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Solicitud APROBADA correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RevertApproval(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.RevertApproval(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error al revertir aprobación: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Estado revertido a SOLICITADO")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) Reject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.Reject(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error al rechazar la solicitud: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Solicitud RECHAZADA")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) Print(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		utils.Render(c, "solicitud/derecho/modal_print", gin.H{
			"Solicitud": solicitud,
		})
		return
	}

	personaView, errMongo := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), solicitud.Usuario.CI)
	if errMongo != nil {
		personaView = nil
	}

	mode := c.Query("mode")
	pdf := ctrl.reportService.GeneratePV01(c.Request.Context(), solicitud, personaView, mode)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV01-%s.pdf", solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudDerechoController) Destroy(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	if err := ctrl.solicitudService.Delete(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error eliminando: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud eliminada")
	c.Redirect(http.StatusFound, fmt.Sprintf("/cupos/derecho/%s", solicitud.UsuarioID))
}

func (ctrl *SolicitudDerechoController) ApproveItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.ApproveItem(c.Request.Context(), id, itemID); err != nil {
		c.String(http.StatusInternalServerError, "Error al aprobar el tramo: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Tramo APROBADO correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RejectItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.RejectItem(c.Request.Context(), id, itemID); err != nil {
		c.String(http.StatusInternalServerError, "Error al rechazar el tramo: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Tramo RECHAZADO")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RevertApprovalItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.solicitudService.RevertApprovalItem(c.Request.Context(), id, itemID); err != nil {
		c.String(http.StatusInternalServerError, "Error al revertir aprobación del tramo: "+err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Aprobación de tramo REVERTIDA")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) GetReprogramarModalSolicitudItem(c *gin.Context) {
	id := c.Param("id")
	item, err := ctrl.solicitudService.GetItemByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Tramo no encontrado")
		return
	}

	utils.Render(c, "solicitud/components/modal_reprogramar_pasaje", gin.H{
		"SolicitudItem": item,
		"Tipo":          item.Tipo,
	})
}

func (ctrl *SolicitudDerechoController) ReprogramarItem(c *gin.Context) {
	var req dtos.ReprogramarSolicitudItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes?error=DatosInvalidos")
		return
	}

	if err := ctrl.solicitudService.ReprogramarItem(c.Request.Context(), req); err != nil {
		utils.SetErrorMessage(c, "Error: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	utils.SetSuccessMessage(c, "Reprogramación de tramo solicitada exitosamente")
	c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
}
