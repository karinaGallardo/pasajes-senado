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

	currentUser := appcontext.CurrentUser(c)

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

	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), item.SenTitularID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Usuario titular del derecho no encontrado")
		return
	}

	canCreate := false
	if currentUser.ID == targetUser.ID {
		canCreate = true
	} else if currentUser.IsAdminOrResponsable() {
		canCreate = true
	} else if targetUser.EncargadoID != nil && *targetUser.EncargadoID == currentUser.ID {
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

	if itinerario.Codigo == "SOLO_IDA" {
		origen = userLoc
		destino = lpbLoc
	} else {
		origen = lpbLoc
		destino = userLoc
	}

	utils.Render(c, "solicitud/derecho/create", gin.H{
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

	usuario := appcontext.CurrentUser(c)

	solicitud, err := ctrl.solicitudService.Create(c.Request.Context(), req, usuario)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creando solicitud: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud creada correctamente")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID))
}

func (ctrl *SolicitudDerechoController) Edit(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No se puede editar una solicitud que no está en estado SOLICITADO")
		return
	}

	currentUser := appcontext.CurrentUser(c)

	if !currentUser.CanEditSolicitud(*solicitud) {
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
			ItinerarioIdaID = t.ID
		}
		if t.Codigo == "SOLO_VUELTA" {
			ItinerarioVueltaID = t.ID
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
	if ActiveTab == "SOLO_IDA" {
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

	utils.Render(c, "solicitud/derecho/edit", gin.H{
		"Aerolineas":         aerolineas,
		"TargetUser":         &solicitud.Usuario,
		"User":               currentUser,
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

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No editable")
		return
	}

	solicitud.TipoSolicitudID = req.TipoSolicitudID
	solicitud.AmbitoViajeID = req.AmbitoViajeID
	if req.TipoItinerarioID != "" {
		solicitud.TipoItinerarioID = req.TipoItinerarioID
	}
	solicitud.OrigenIATA = req.OrigenIATA
	solicitud.DestinoIATA = req.DestinoIATA
	solicitud.FechaIda = fechaIda
	solicitud.FechaVuelta = fechaVuelta
	solicitud.Motivo = req.Motivo
	solicitud.AerolineaSugerida = req.AerolineaSugerida
	if err := ctrl.solicitudService.Update(c.Request.Context(), solicitud); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud actualizada correctamente")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID))
}

func (ctrl *SolicitudDerechoController) Show(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)

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

	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())

	utils.Render(c, "solicitud/derecho/show", gin.H{
		"Title":        "Detalle Solicitud (Derecho) #" + id,
		"Solicitud":    solicitud,
		"Usuarios":     usuariosMap,
		"Step1":        step1,
		"Step2":        step2,
		"Step3":        step3,
		"MermaidGraph": mermaidGraph,
		"Aerolineas":   aerolineas,
		"Rutas":        rutas,
		"Agencias":     agencias,
	})
}

func (ctrl *SolicitudDerechoController) Approve(c *gin.Context) {
	id := c.Param("id")
	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil || !currentUser.CanApproveReject() {
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

func (ctrl *SolicitudDerechoController) Reject(c *gin.Context) {
	id := c.Param("id")
	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil || !currentUser.CanApproveReject() {
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

	pdf := ctrl.reportService.GeneratePV01(c.Request.Context(), solicitud, personaView)

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
