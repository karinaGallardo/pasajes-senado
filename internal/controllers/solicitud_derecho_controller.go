package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudDerechoController struct {
	solicitudService        *services.SolicitudService
	solicitudDerechoService *services.SolicitudDerechoService
	destinoService          *services.DestinoService
	conceptoService         *services.ConceptoService
	tipoSolicitudService    *services.TipoSolicitudService
	ambitoService           *services.AmbitoService
	cupoService             *services.CupoService
	userService             *services.UsuarioService
	peopleService           *services.PeopleService
	reportService           *services.ReportService
	aerolineaService        *services.AerolineaService
	agenciaService          *services.AgenciaService
	tipoItinerarioService   *services.TipoItinerarioService
	rutaService             *services.RutaService
	descargoService         *services.DescargoService
	configuracionService    *services.ConfiguracionService
	openTicketService       *services.OpenTicketService
}

func NewSolicitudDerechoController(
	solicitudService *services.SolicitudService,
	solicitudDerechoService *services.SolicitudDerechoService,
	destinoService *services.DestinoService,
	conceptoService *services.ConceptoService,
	tipoSolicitudService *services.TipoSolicitudService,
	ambitoService *services.AmbitoService,
	cupoService *services.CupoService,
	userService *services.UsuarioService,
	peopleService *services.PeopleService,
	reportService *services.ReportService,
	aerolineaService *services.AerolineaService,
	agenciaService *services.AgenciaService,
	tipoItinerarioService *services.TipoItinerarioService,
	rutaService *services.RutaService,
	descargoService *services.DescargoService,
	configuracionService *services.ConfiguracionService,
	openTicketService *services.OpenTicketService,
) *SolicitudDerechoController {
	return &SolicitudDerechoController{
		solicitudService:        solicitudService,
		solicitudDerechoService: solicitudDerechoService,
		destinoService:          destinoService,
		conceptoService:         conceptoService,
		tipoSolicitudService:    tipoSolicitudService,
		ambitoService:           ambitoService,
		cupoService:             cupoService,
		userService:             userService,
		peopleService:           peopleService,
		reportService:           reportService,
		aerolineaService:        aerolineaService,
		agenciaService:          agenciaService,
		tipoItinerarioService:   tipoItinerarioService,
		rutaService:             rutaService,
		descargoService:         descargoService,
		configuracionService:    configuracionService,
		openTicketService:       openTicketService,
	}
}

func (ctrl *SolicitudDerechoController) GetCreateModal(c *gin.Context) {
	itemID := c.Param("item_id")
	cupoItem, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	targetUser, _ := ctrl.userService.GetByID(c.Request.Context(), cupoItem.SenAsignadoID)

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	tipoSolicitud, ambitoNac, _ := ctrl.tipoSolicitudService.GetByCodigoAndAmbito(c.Request.Context(), "USO_CUPO", "NACIONAL")

	var origen, destino *models.Destino
	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), targetUser.GetOrigenIATA())
	sedesAutorizadas := ctrl.getSedesAutorizadas(c.Request.Context())
	var sedeDefault *models.Destino
	if len(sedesAutorizadas) > 0 {
		sedeDefault = &sedesAutorizadas[0]
	}

	// Default orientation: from Senator's origin to first authorized sede
	origen = userLoc
	destino = sedeDefault

	// Preparar Orígenes Autorizados para el Senador
	origenesAutorizados := []models.Destino{}
	if userLoc != nil {
		origenesAutorizados = append(origenesAutorizados, *userLoc)
	}
	for _, alt := range targetUser.OrigenesAlternativos {
		if alt.Destino != nil {
			origenesAutorizados = append(origenesAutorizados, *alt.Destino)
		}
	}

	referer := c.Request.Referer()
	if referer == "" {
		referer = "/cupos/derecho/" + targetUser.ID
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
		"DefaultDateIda":      time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		"DefaultDateVuelta":   time.Now().AddDate(0, 0, 4).Format("2006-01-02"),
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

	// Cargar datos complementarios
	cupoItem, _ := ctrl.cupoService.GetCupoDerechoItemByID(c.Request.Context(), *solicitud.CupoDerechoItemID)
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	sedesAutorizadas := ctrl.getSedesAutorizadas(c.Request.Context())
	referer := c.Request.Referer()

	var sedeDefault *models.Destino
	if len(sedesAutorizadas) > 0 {
		sedeDefault = &sedesAutorizadas[0]
	}

	// Hidratar permisos de los ítems una sola vez
	for i := range solicitud.Items {
		solicitud.Items[i].HydratePermissions(authUser)
	}

	userLoc, _ := ctrl.destinoService.GetByIATA(c.Request.Context(), solicitud.Usuario.GetOrigenIATA())
	ida := solicitud.GetItemIda()
	vuelta := solicitud.GetItemVuelta()
	itin := solicitud.TipoItinerario

	// Determinar Origen, Destino y Sede con fallbacks limpios
	var origen, destino, sede *models.Destino

	if ida != nil {
		origen = ida.Origen
		sede = ida.Destino
	}

	if vuelta != nil {
		if origen == nil {
			origen = vuelta.Destino
		}
		destino = vuelta.Destino
		if sede == nil {
			sede = vuelta.Origen
		}
	}

	// Fallback final para visualización inicial
	if origen == nil {
		origen = userLoc
	}

	if destino == nil {
		if vuelta != nil && vuelta.Destino != nil {
			destino = vuelta.Destino
		} else if ida != nil && ida.Destino != nil {
			destino = ida.Destino
		} else {
			destino = sedeDefault
		}
	}

	if sede == nil {
		sede = sedeDefault
	}

	// Orígenes alternativos autorizados
	origenesAutorizados := []models.Destino{}
	if userLoc != nil {
		origenesAutorizados = append(origenesAutorizados, *userLoc)
	}
	for _, alt := range solicitud.Usuario.OrigenesAlternativos {
		if alt.Destino != nil {
			origenesAutorizados = append(origenesAutorizados, *alt.Destino)
		}
	}

	// Calcular banderas especializadas para solicitudes de Derecho (Ida/Vuelta únicas)
	canEditIda := (ida != nil && ida.CanEdit())
	canEditVuelta := (vuelta != nil && vuelta.CanEdit())
	isIdaProgramada := (ida != nil && !ida.IsPendiente())
	isVueltaProgramada := (vuelta != nil && !vuelta.IsPendiente())

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
		"Origen":              origen,
		"Destino":             destino,
		"Sede":                sede,
		"SedesAutorizadas":    sedesAutorizadas,
		"OrigenesAutorizados": origenesAutorizados,
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
	c.Header("HX-Trigger", "solicitudUpdated") // Opcional si necesitas disparar algún evento

	// If HTMX request, re-render the edit modal with updated data
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

	// --- 2. Stepper & Status Logic delegated to Model ---
	steps, showNextSteps := solicitud.GetStepperData()
	statusCard := solicitud.GetStatusCardData()

	// dependencies
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
		c.String(http.StatusForbidden, "No tiene permisos para aprobar esta solicitud en su estado actual")
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRevertApproval {
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
		c.String(http.StatusForbidden, "No tiene permisos para rechazar esta solicitud en su estado actual")
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanDelete {
		c.String(http.StatusForbidden, "No tiene permisos para eliminar esta solicitud")
		return
	}

	if err := ctrl.solicitudService.Delete(c.Request.Context(), id, authUser.ID); err != nil {
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	item := solicitud.GetItemByID(itemID)
	if item == nil {
		c.String(http.StatusNotFound, "Tramo no encontrado")
		return
	}

	item.HydratePermissions(authUser)
	if !item.Permissions.CanRevert {
		c.String(http.StatusForbidden, "No tiene permisos para revertir la aprobación de este tramo")
		return
	}

	if err := ctrl.solicitudService.RevertApprovalItem(c.Request.Context(), id, itemID); err != nil {
		c.String(http.StatusInternalServerError, "Error al revertir aprobación del tramo: "+err.Error())
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

func (ctrl *SolicitudDerechoController) getSedesAutorizadas(ctx context.Context) []models.Destino {
	sedesAutorizadas := []models.Destino{}

	// Obtener configuración de sedes (ej: "LPB,SRE,TJA")
	sedesStr := ctrl.configuracionService.GetValue(ctx, "SEDES_AUTORIZADAS")
	var sedesCodes []string

	if sedesStr == "" {
		// Fallback por defecto si no hay configuración
		sedesCodes = []string{"LPB"}
	} else {
		// Limpiar y separar códigos
		parts := strings.Split(sedesStr, ",")
		for _, p := range parts {
			code := strings.ToUpper(strings.TrimSpace(p))
			if code != "" {
				sedesCodes = append(sedesCodes, code)
			}
		}
	}

	// Buscar destinos correspondientes
	for _, code := range sedesCodes {
		if s, err := ctrl.destinoService.GetByIATA(ctx, code); err == nil && s != nil {
			sedesAutorizadas = append(sedesAutorizadas, *s)
		}
	}

	return sedesAutorizadas
}
