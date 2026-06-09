package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type SolicitudDerechoController struct {
	solicitudService        *services.SolicitudService
	solicitudDerechoService *services.SolicitudDerechoService
	destinoService          *services.DestinoService
	tipoSolicitudService    *services.TipoSolicitudService
	cupoService             *services.CupoService
	userService             *services.UsuarioService
	peopleService           *services.PeopleService
	reportService           *services.ReportService
	aerolineaService        *services.AerolineaService
	descargoService         *services.DescargoService
	openTicketService       *services.OpenTicketService
}

func NewSolicitudDerechoController(
	solicitudService *services.SolicitudService,
	solicitudDerechoService *services.SolicitudDerechoService,
	destinoService *services.DestinoService,
	tipoSolicitudService *services.TipoSolicitudService,
	cupoService *services.CupoService,
	userService *services.UsuarioService,
	peopleService *services.PeopleService,
	reportService *services.ReportService,
	aerolineaService *services.AerolineaService,
	descargoService *services.DescargoService,
	openTicketService *services.OpenTicketService,
) *SolicitudDerechoController {
	return &SolicitudDerechoController{
		solicitudService:        solicitudService,
		solicitudDerechoService: solicitudDerechoService,
		destinoService:          destinoService,
		tipoSolicitudService:    tipoSolicitudService,
		cupoService:             cupoService,
		userService:             userService,
		peopleService:           peopleService,
		reportService:           reportService,
		aerolineaService:        aerolineaService,
		descargoService:         descargoService,
		openTicketService:       openTicketService,
	}
}

func (ctrl *SolicitudDerechoController) GetCreateModal(c *gin.Context) {
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	cupoItem, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	targetUser, _ := ctrl.userService.GetByID(c.Request.Context(), cupoItem.SenAsignadoID)

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	tipoSolicitud, ambitoNac, _ := ctrl.tipoSolicitudService.GetByCodigoAndAmbito(c.Request.Context(), "USO_CUPO", "NACIONAL")

	var origen, destino *models.Destino
	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), targetUser.GetOrigenIATA())
	sedesAutorizadas := ctrl.solicitudDerechoService.GetSedesAutorizadas(c.Request.Context())
	var sedeDefault *models.Destino
	if len(sedesAutorizadas) > 0 {
		sedeDefault = &sedesAutorizadas[0]
	}

	origen = userLoc
	destino = sedeDefault

	origenesAutorizados := ctrl.userService.BuildOrigenesAutorizados(targetUser, userLoc)

	referer := c.Request.Referer()
	if referer == "" {
		referer = "/cupos/derecho/" + targetUser.ID
	}

	defaultIda, defaultVuelta := ctrl.solicitudDerechoService.GetDefaultTravelDates()
	openTickets, _ := ctrl.openTicketService.GetDisponiblesByUsuarioID(c.Request.Context(), targetUser.ID)
	allDestinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	var destinos []models.Destino
	for _, d := range allDestinos {
		if d.AmbitoCodigo == "NACIONAL" {
			destinos = append(destinos, d)
		}
	}

	utils.Render(c, "solicitud/derecho/modal_create", gin.H{
		"TargetUser":          targetUser,
		"Aerolineas":          aerolineas,
		"CupoItem":            cupoItem,
		"Concepto":            tipoSolicitud.ConceptoViaje,
		"TipoSolicitud":       tipoSolicitud,
		"Ambito":              ambitoNac,
		"Origen":              origen,
		"Destino":             destino,
		"OrigenesAutorizados": origenesAutorizados,
		"SedesAutorizadas":    sedesAutorizadas,
		"Sede":                sedeDefault,
		"ReturnURL":           referer,
		"DefaultDateIda":      defaultIda,
		"DefaultDateVuelta":   defaultVuelta,
		"OpenTickets":         openTickets,
		"CanManageSystem":     authUser.IsAdminOrResponsable(),
		"AuthUser":            authUser,
		"Destinos":            destinos,
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

	solicitud, err := ctrl.solicitudDerechoService.CreateDerecho(c.Request.Context(), req, authUser)
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

func (ctrl *SolicitudDerechoController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	authUser := appcontext.AuthUser(c)
	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanEdit {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	cupoItem, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), *solicitud.CupoDerechoItemID)
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	sedesAutorizadas := ctrl.solicitudDerechoService.GetSedesAutorizadas(c.Request.Context())
	referer := c.Request.Referer()

	for i := range solicitud.Items {
		solicitud.Items[i].HydratePermissions(authUser)
	}

	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), solicitud.Usuario.GetOrigenIATA())
	defaults := ctrl.solicitudDerechoService.GetEditFormDefaults(c.Request.Context(), solicitud, userLoc)

	origenesAutorizados := ctrl.userService.BuildOrigenesAutorizados(&solicitud.Usuario, userLoc)

	ida := solicitud.GetItemIda()
	vuelta := solicitud.GetItemVuelta()
	itin := solicitud.TipoItinerario
	canEditIda := (ida != nil && ida.CanEdit())
	canEditVuelta := (vuelta != nil && vuelta.CanEdit())
	isIdaProgramada := (ida != nil && !ida.IsPendiente())
	isVueltaProgramada := (vuelta != nil && !vuelta.IsPendiente())

	openTickets, _ := ctrl.openTicketService.GetDisponiblesByUsuarioID(c.Request.Context(), solicitud.UsuarioID)

	utils.Render(c, "solicitud/derecho/modal_edit", gin.H{
		"Aerolineas":          aerolineas,
		"TargetUser":          &solicitud.Usuario,
		"CupoItem":            cupoItem,
		"Solicitud":           solicitud,
		"CanEditIda":          canEditIda,
		"CanEditVuelta":       canEditVuelta,
		"IsIdaProgramada":     isIdaProgramada,
		"IsVueltaProgramada":  isVueltaProgramada,
		"Concepto":            solicitud.TipoSolicitud.ConceptoViaje,
		"TipoSolicitud":       solicitud.TipoSolicitud,
		"Ambito":              solicitud.AmbitoViaje,
		"Itinerario":          itin,
		"ReturnURL":           referer,
		"Origen":              defaults.Origen,
		"Destino":             defaults.Destino,
		"Sede":                defaults.Sede,
		"SedesAutorizadas":    sedesAutorizadas,
		"OrigenesAutorizados": origenesAutorizados,
		"OpenTickets":         openTickets,
	})
}

func (ctrl *SolicitudDerechoController) Update(c *gin.Context) {
	id := c.Param("id")
	var req dtos.UpdateSolicitudRequest

	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudDerechoService.UpdateDerecho(c.Request.Context(), id, req, authUser)
	if err != nil {
		c.String(http.StatusBadRequest, "Error: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Solicitud actualizada correctamente")
	c.Header("HX-Trigger", "solicitudUpdated")

	if c.GetHeader("HX-Request") == "true" {
		ctrl.GetEditModal(c)
		return
	}

	targetURL := fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitud.ID)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}
	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *SolicitudDerechoController) Show(c *gin.Context) {
	authUser := appcontext.AuthUser(c)

	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	if !solicitud.CanView(authUser) {
		c.String(http.StatusForbidden, "No tiene permiso para ver esta solicitud")
		return
	}
	solicitud.HydratePermissions(authUser)

	steps, showNextSteps := solicitud.GetStepperData()
	statusCard := solicitud.GetStatusCardData()

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(c.Request.Context(), []models.Solicitud{*solicitud})

	var descargoID string
	if descargo, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), id); descargo != nil && descargo.ID != "" {
		descargoID = descargo.ID
	}

	utils.Render(c, "solicitud/derecho/show", gin.H{
		"Title":         "Detalle Solicitud (Derecho) #" + id,
		"Solicitud":     solicitud,
		"Usuarios":      usuariosMap,
		"DescargoID":    descargoID,
		"Steps":         steps,
		"ShowNextSteps": showNextSteps,
		"StatusCard":    statusCard,
		"Aerolineas":    aerolineas,
	})
}

func (ctrl *SolicitudDerechoController) Approve(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.Approve(c.Request.Context(), id, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Solicitud APROBADA correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RevertApproval(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.RevertApproval(c.Request.Context(), id, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Estado revertido a SOLICITADO")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) Reject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.Reject(c.Request.Context(), id, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Solicitud RECHAZADA")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RevertReject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.RevertReject(c.Request.Context(), id, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Estado revertido a SOLICITADO")
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

	disposition := "inline"
	if utils.IsMobileBrowser(c) {
		disposition = "attachment"
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"FORM-PV01-%s.pdf\"", disposition, solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudDerechoController) Destroy(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	userID := "" // fallback
	if sol, _ := ctrl.solicitudService.GetByID(c.Request.Context(), id); sol != nil {
		userID = sol.UsuarioID
	}
	if err := ctrl.solicitudService.Delete(c.Request.Context(), id, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Solicitud eliminada")
	c.Redirect(http.StatusFound, fmt.Sprintf("/cupos/derecho/%s", userID))
}

func (ctrl *SolicitudDerechoController) ApproveItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.ApproveItem(c.Request.Context(), id, itemID, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Tramo APROBADO correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RejectItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.RejectItem(c.Request.Context(), id, itemID, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Tramo RECHAZADO")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) RevertApprovalItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	if err := ctrl.solicitudService.RevertApprovalItem(c.Request.Context(), id, itemID, authUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}
	utils.SetSuccessMessage(c, "Aprobación de tramo REVERTIDA")
	c.Redirect(http.StatusFound, "/solicitudes/derecho/"+id+"/detalle")
}

func (ctrl *SolicitudDerechoController) GetItemRegularizacionModal(c *gin.Context) {
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	item, err := ctrl.solicitudService.GetItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(404, "Tramo no encontrado")
		return
	}

	item.Solicitud.HydratePermissions(authUser)
	if !item.Solicitud.Permissions.CanRegularize {
		c.String(403, "No tiene permisos para regularizar fechas")
		return
	}

	utils.Render(c, "solicitud/derecho/modal_regularizacion_item", gin.H{
		"Item":      item,
		"Solicitud": item.Solicitud,
	})
}

func (ctrl *SolicitudDerechoController) UpdateItemRegularizacionDates(c *gin.Context) {
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	item, err := ctrl.solicitudService.GetItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(404, "Tramo no encontrado")
		return
	}

	item.Solicitud.HydratePermissions(authUser)
	if !item.Solicitud.Permissions.CanRegularize {
		c.String(403, "No tiene permisos para realizar esta acción")
		return
	}

	fecha := c.PostForm("fecha_regularizacion")

	if err := ctrl.solicitudService.UpdateSolicitudItemDates(c.Request.Context(), itemID, fecha, fecha); err != nil {
		c.String(400, err.Error())
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(200)
}
