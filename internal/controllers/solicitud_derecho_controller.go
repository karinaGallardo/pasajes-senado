package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudDerechoController struct {
	solicitudService      *services.SolicitudService
	ciudadService         *services.CiudadService
	conceptoService       *services.ConceptoService
	tipoSolicitudService  *services.TipoSolicitudService
	ambitoService         *services.AmbitoService
	cupoService           *services.CupoService
	userService           *services.UsuarioService
	peopleService         *services.PeopleService
	reportService         *services.ReportService
	aerolineaService      *services.AerolineaService
	tipoItinerarioService *services.TipoItinerarioService
}

func NewSolicitudDerechoController() *SolicitudDerechoController {
	return &SolicitudDerechoController{
		solicitudService:      services.NewSolicitudService(),
		ciudadService:         services.NewCiudadService(),
		conceptoService:       services.NewConceptoService(),
		tipoSolicitudService:  services.NewTipoSolicitudService(),
		ambitoService:         services.NewAmbitoService(),
		cupoService:           services.NewCupoService(),
		userService:           services.NewUsuarioService(),
		peopleService:         services.NewPeopleService(),
		reportService:         services.NewReportService(),
		aerolineaService:      services.NewAerolineaService(),
		tipoItinerarioService: services.NewTipoItinerarioService(),
	}
}

func (ctrl *SolicitudDerechoController) Create(c *gin.Context) {
	destinos, _ := ctrl.ciudadService.GetAll()
	conceptos, _ := ctrl.conceptoService.GetAll()

	currentUser := appcontext.CurrentUser(c)

	voucherID := c.Param("voucher_id")
	itinerarioCode := c.Param("itinerario_code")

	voucher, err := ctrl.cupoService.GetVoucherByID(voucherID)
	if err != nil {
		c.String(http.StatusNotFound, "Voucher no encontrada")
		return
	}

	itinerario, err := ctrl.tipoItinerarioService.GetByCodigo(itinerarioCode)
	if err != nil {
		c.String(http.StatusBadRequest, "Itinerario inválido")
		return
	}

	targetUser, err := ctrl.userService.GetByID(voucher.SenadorID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Usuario titular del voucher no encontrado")
		return
	}

	canCreate := false
	if currentUser.ID == targetUser.ID {
		canCreate = true
	} else if currentUser.Rol.Codigo == "ADMIN" || currentUser.Rol.Codigo == "TECNICO" {
		canCreate = true
	} else if targetUser.EncargadoID != nil && *targetUser.EncargadoID == currentUser.ID {
		canCreate = true
	}

	if !canCreate {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta solicitud")
		return
	}

	var alertaOrigen string
	if targetUser.GetOrigenCode() == "" {
		alertaOrigen = "Este usuario no tiene configurado su LUGAR DE ORIGEN en el perfil. El sistema no podrá calcular rutas automáticamente."
	}

	aereos, _ := ctrl.aerolineaService.GetAllActive()
	var aerolineas []string
	for _, a := range aereos {
		aerolineas = append(aerolineas, a.Nombre)
	}

	tipoSolicitud, ambitoNac, _ := ctrl.tipoSolicitudService.GetByCodigoAndAmbito("USO_CUPO", "NACIONAL")

	var weekDays []map[string]string
	dayNames := map[string]string{
		"Monday":    "Lun",
		"Tuesday":   "Mar",
		"Wednesday": "Mie",
		"Thursday":  "Jue",
		"Friday":    "Vie",
		"Saturday":  "Sab",
		"Sunday":    "Dom",
	}

	if voucher.FechaDesde != nil && voucher.FechaHasta != nil {
		for d := *voucher.FechaDesde; !d.After(*voucher.FechaHasta); d = d.AddDate(0, 0, 1) {
			esName := dayNames[d.Weekday().String()]
			weekDays = append(weekDays, map[string]string{
				"date":   d.Format("2006-01-02"),
				"name":   esName,
				"dayNum": d.Format("02"),
			})
		}
	}

	userOrigenCode := targetUser.GetOrigenCode()
	userOrigenName := "Ciudad No Configurada"
	if targetUser.Origen != nil {
		userOrigenName = targetUser.Origen.Nombre
	}
	lpbCode := "LPB"
	lpbName := "La Paz - El Alto"

	var origenCode, origenName, origenLabel string
	var destinoCode, destinoName, destinoLabel string

	if itinerario.Codigo == "SOLO_VUELTA" {
		origenCode = lpbCode
		origenName = lpbName
		origenLabel = "Sede de Funciones"

		destinoCode = userOrigenCode
		destinoName = userOrigenName
		destinoLabel = "Origen de " + targetUser.Firstname
	} else {
		origenCode = userOrigenCode
		origenName = userOrigenName
		origenLabel = "Origen de " + targetUser.Firstname

		destinoCode = lpbCode
		destinoName = lpbName
		destinoLabel = "Sede de Funciones"
	}

	utils.Render(c, "solicitud/derecho/create.html", gin.H{
		"Title":        "Pasaje por Derecho - " + targetUser.GetNombreCompleto(),
		"TargetUser":   targetUser,
		"Destinos":     destinos,
		"Conceptos":    conceptos,
		"Aerolineas":   aerolineas,
		"AlertaOrigen": alertaOrigen,
		"Voucher":      voucher,
		"WeekDays":     weekDays,

		"Concepto":      tipoSolicitud.ConceptoViaje,
		"TipoSolicitud": tipoSolicitud,
		"Ambito":        ambitoNac,

		"Itinerario":   itinerario,
		"OrigenCode":   origenCode,
		"OrigenName":   origenName,
		"OrigenLabel":  origenLabel,
		"DestinoCode":  destinoCode,
		"DestinoName":  destinoName,
		"DestinoLabel": destinoLabel,
	})
}

func (ctrl *SolicitudDerechoController) Store(c *gin.Context) {
	var req dtos.CreateSolicitudRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	if req.VoucherID == "" {
		c.String(http.StatusBadRequest, "Voucher ID requerido para solicitud de derecho")
		return
	}

	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" {
		if t, err := time.Parse(layout, req.FechaVuelta); err == nil {
			fechaVuelta = &t
		}
	}

	usuario := appcontext.CurrentUser(c)

	var realSolicitanteID string
	if req.TargetUserID != "" {
		realSolicitanteID = req.TargetUserID
	} else {
		realSolicitanteID = usuario.ID
	}

	itinID := req.TipoItinerarioID
	if itinID == "" {
		itin, _ := ctrl.tipoItinerarioService.GetByCodigo("IDA_VUELTA")
		if itin != nil {
			itinID = itin.ID
		}
	}

	nuevaSolicitud := models.Solicitud{
		UsuarioID:         realSolicitanteID,
		TipoSolicitudID:   req.TipoSolicitudID,
		AmbitoViajeID:     req.AmbitoViajeID,
		TipoItinerarioID:  itinID,
		OrigenCode:        req.OrigenCode,
		DestinoCode:       req.DestinoCode,
		FechaIda:          fechaIda,
		FechaVuelta:       fechaVuelta,
		Motivo:            req.Motivo,
		AerolineaSugerida: req.AerolineaSugerida,
	}

	if err := ctrl.solicitudService.Create(&nuevaSolicitud, usuario, req.VoucherID); err != nil {
		c.String(http.StatusInternalServerError, "Error creando solicitud: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", nuevaSolicitud.ID))
}

func (ctrl *SolicitudDerechoController) Edit(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.FindByID(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No se puede editar una solicitud que no está en estado SOLICITADO")
		return
	}

	currentUser := appcontext.CurrentUser(c)

	if solicitud.UsuarioID != currentUser.ID && currentUser.Rol.Codigo != "ADMIN" {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	if solicitud.VoucherID == nil {
		c.String(http.StatusBadRequest, "Esta solicitud no corresponde a un pasaje por derecho")
		return
	}

	voucher, err := ctrl.cupoService.GetVoucherByID(*solicitud.VoucherID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Voucher asociado no encontrado")
		return
	}

	TiposItinerario, _ := ctrl.tipoItinerarioService.GetAll()
	var ItinerarioIdaID string
	var ItinerarioVueltaID string
	var ActiveTab string

	for _, t := range TiposItinerario {
		if t.Codigo == "SOLO_IDA" {
			ItinerarioIdaID = t.ID
		}
		if t.Codigo == "SOLO_VUELTA" {
			ItinerarioVueltaID = t.ID
		}
	}

	if solicitud.TipoItinerario != nil {
		ActiveTab = solicitud.TipoItinerario.Codigo
	}

	aereos, _ := ctrl.aerolineaService.GetAllActive()
	var aerolineas []string
	for _, a := range aereos {
		aerolineas = append(aerolineas, a.Nombre)
	}

	utils.Render(c, "solicitud/derecho/edit.html", gin.H{
		"Aerolineas":         aerolineas,
		"TargetUser":         &solicitud.Usuario,
		"User":               currentUser,
		"Itinerarios":        TiposItinerario,
		"ItinerarioIdaID":    ItinerarioIdaID,
		"ItinerarioVueltaID": ItinerarioVueltaID,
		"Voucher":            voucher,
		"VoucherID":          voucher.ID,
		"ActiveTab":          ActiveTab,
		"Solicitud":          solicitud,
		"IsEdit":             true,
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

	solicitud, err := ctrl.solicitudService.FindByID(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No editable")
		return
	}

	solicitud.TipoSolicitudID = req.TipoSolicitudID
	solicitud.AmbitoViajeID = req.AmbitoViajeID
	if req.TipoItinerarioID != "" {
		solicitud.TipoItinerarioID = req.TipoItinerarioID
	}
	solicitud.OrigenCode = req.OrigenCod
	solicitud.DestinoCode = req.DestinoCod
	solicitud.FechaIda = fechaIda
	solicitud.FechaVuelta = fechaVuelta
	solicitud.Motivo = req.Motivo
	solicitud.AerolineaSugerida = req.AerolineaSugerida
	if err := ctrl.solicitudService.Update(solicitud); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID))
}

func (ctrl *SolicitudDerechoController) Show(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.FindByID(id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	st := "SOLICITADO"
	if solicitud.EstadoSolicitudCodigo != nil {
		st = *solicitud.EstadoSolicitudCodigo
	}
	step1 := true
	step2 := st == "APROBADO" || st == "FINALIZADO"
	step3 := st == "FINALIZADO"

	mermaidGraph := "graph TD; A[Registro Solicitud] --> B{¿Autorización?}; B -- Aprobado --> C[Gestión Pasajes]; C --> D[Viaje / Finalizado]; B -- Rechazado --> E[Solicitud Rechazada];\n"
	mermaidGraph += "classDef default fill:#fff,stroke:#333,stroke-width:1px; classDef active fill:#03738C,stroke:#03738C,stroke-width:2px,color:#fff;\n"

	switch st {
	case "SOLICITADO":
		mermaidGraph += "class A active;"
	case "APROBADO":
		mermaidGraph += "class C active;"
	case "FINALIZADO":
		mermaidGraph += "class D active;"
	case "RECHAZADO":
		mermaidGraph += "class E active;"
	}

	aerolineas := []string{"BoA - Boliviana de Aviación", "EcoJet"}

	var creator *models.Usuario
	if solicitud.CreatedBy != nil {
		creator, _ = ctrl.userService.GetByID(*solicitud.CreatedBy)
	}

	utils.Render(c, "solicitud/derecho/show.html", gin.H{
		"Title":        "Detalle Solicitud (Derecho) #" + id,
		"Solicitud":    solicitud,
		"Creator":      creator,
		"Step1":        step1,
		"Step2":        step2,
		"Step3":        step3,
		"MermaidGraph": mermaidGraph,
		"Aerolineas":   aerolineas,
		"User":         appcontext.CurrentUser(c),
	})
}

func (ctrl *SolicitudDerechoController) Approve(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.solicitudService.Approve(id); err != nil {
		// log.Printf("Error approving solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) Reject(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.solicitudService.Reject(id); err != nil {
		// log.Printf("Error rejecting solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) Print(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.FindByID(id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	personaView, errMongo := ctrl.peopleService.FindSenatorDataByCI(solicitud.Usuario.CI)
	if errMongo != nil {
		personaView = nil
	}

	pdf := ctrl.reportService.GeneratePV01(solicitud, personaView)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV01-%s.pdf", solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudDerechoController) Destroy(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.FindByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	if err := ctrl.solicitudService.Delete(id); err != nil {
		c.String(http.StatusInternalServerError, "Error eliminando: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/cupos/derecho/%s", solicitud.UsuarioID))
}
