package controllers

import (
	"context"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	service           *services.SolicitudService
	userService       *services.UsuarioService
	openTicketService *services.OpenTicketService
}

func NewSolicitudController(service *services.SolicitudService, userService *services.UsuarioService, openTicketService *services.OpenTicketService) *SolicitudController {
	return &SolicitudController{
		service:           service,
		userService:       userService,
		openTicketService: openTicketService,
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	c.Redirect(302, "/solicitudes/derecho")
}

func (ctrl *SolicitudController) IndexPendientesDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPendientesDescargoPaginated(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		// handle empty
	}

	solicitudes := result.Solicitudes

	// Get users for map
	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}
	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}
	usuariosMap := make(map[string]*models.Usuario)
	if len(ids) > 0 {
		usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
		for i := range usuarios {
			usuariosMap[usuarios[i].ID] = &usuarios[i]
		}
	}

	// Stats for sidebar/topbar
	pendingRequests, _ := ctrl.service.GetPendingCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())
	pendingDescargosCount, _ := ctrl.service.GetPendientesDescargo(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())
	var userIDs []string
	if !authUser.IsAdminOrResponsable() {
		userIDs = []string{authUser.ID}
		if senators, err := ctrl.userService.GetSenatorsByEncargado(c.Request.Context(), authUser.ID); err == nil {
			for _, s := range senators {
				userIDs = append(userIDs, s.ID)
			}
		}
	}
	openTicketCount := ctrl.openTicketService.GetPendingCount(c.Request.Context(), userIDs)
	openTicketDescargoCount := ctrl.service.GetConDescargoOpenTicketCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())

	utils.Render(c, "solicitud/pendientes_descargo", gin.H{
		"Title":                   "Solicitudes Pendientes de Descargo",
		"Result":                  result,
		"Usuarios":                usuariosMap,
		"PendientesDescargo":      true,
		"LinkBase":                "/solicitudes/pendientes-descargo",
		"PendingRequests":         pendingRequests,
		"PendingDescargos":        len(pendingDescargosCount),
		"OpenTicketCount":         openTicketCount,
		"OpenTicketDescargoCount": openTicketDescargoCount,
		"TotalPending":            pendingRequests + int64(len(pendingDescargosCount)) + openTicketCount + openTicketDescargoCount,
	})
}

func (ctrl *SolicitudController) TablePendientesDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPendientesDescargoPaginated(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		// handle empty
	}

	solicitudes := result.Solicitudes

	// Get users for map
	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}
	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}
	usuariosMap := make(map[string]*models.Usuario)
	if len(ids) > 0 {
		usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
		for i := range usuarios {
			usuariosMap[usuarios[i].ID] = &usuarios[i]
		}
	}

	utils.Render(c, "solicitud/table_pendientes_descargo", gin.H{
		"Result":             result,
		"Usuarios":           usuariosMap,
		"PendientesDescargo": true,
		"LinkBase":           "/solicitudes/pendientes-descargo",
	})
}

func (ctrl *SolicitudController) IndexDerecho(c *gin.Context) {
	ctrl.renderIndex(c, "DERECHO", "Solicitudes por Derecho", "solicitud/index")
}

func (ctrl *SolicitudController) IndexOficial(c *gin.Context) {
	ctrl.renderIndex(c, "OFICIAL", "Solicitudes Oficiales", "solicitud/index")
}

func (ctrl *SolicitudController) TableDerecho(c *gin.Context) {
	ctrl.renderIndex(c, "DERECHO", "Solicitudes por Derecho", "solicitud/table_solicitudes")
}

func (ctrl *SolicitudController) TableOficial(c *gin.Context) {
	ctrl.renderIndex(c, "OFICIAL", "Solicitudes Oficiales", "solicitud/table_solicitudes")
}

func (ctrl *SolicitudController) renderIndex(c *gin.Context, concepto string, title string, template string) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	status := c.Query("estado")
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPaginated(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable(), status, concepto, page, limit, searchTerm)
	if err != nil {
		// handle or initialize empty result
	}

	solicitudes := result.Solicitudes

	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}

	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}

	usuariosMap := make(map[string]*models.Usuario)
	if len(ids) > 0 {
		usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
		for i := range usuarios {
			usuariosMap[usuarios[i].ID] = &usuarios[i]
		}
	}

	// Hydrate permissions for all items in the list
	for i := range solicitudes {
		solicitudes[i].HydratePermissions(authUser)
	}

	linkBase := "/solicitudes"
	switch concepto {
	case "DERECHO":
		linkBase = "/solicitudes/derecho"
	case "OFICIAL":
		linkBase = "/solicitudes/oficial"
	}

	utils.Render(c, template, gin.H{
		"Title":             title,
		"Result":            result,
		"Usuarios":          usuariosMap,
		"Status":            status,
		"Concepto":          concepto,
		"LinkBase":          linkBase,
		"AvailableStatuses": models.GetAvailableStatuses(),
	})
}

func (ctrl *SolicitudController) GetPendingStats(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Status(401)
		return
	}

	pendingRequests, _ := ctrl.service.GetPendingCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())
	pendingDescargos, _ := ctrl.service.GetPendientesDescargo(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())

	// Open Tickets count (PENDIENTE states for badge)
	var userIDs []string
	if !authUser.IsAdminOrResponsable() {
		userIDs = []string{authUser.ID}
		if senators, err := ctrl.userService.GetSenatorsByEncargado(c.Request.Context(), authUser.ID); err == nil {
			for _, s := range senators {
				userIDs = append(userIDs, s.ID)
			}
		}
	}
	openTicketCount := ctrl.openTicketService.GetPendingCount(c.Request.Context(), userIDs)
	openTicketDescargoCount := ctrl.service.GetConDescargoOpenTicketCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())

	utils.Render(c, "layouts/components/pending_stats", gin.H{
		"PendingRequests":         pendingRequests,
		"PendingDescargos":        len(pendingDescargos),
		"OpenTicketCount":         openTicketCount,
		"OpenTicketDescargoCount": openTicketDescargoCount,
		"TotalPending":            pendingRequests + int64(len(pendingDescargos)) + openTicketCount + openTicketDescargoCount,
	})
}

func (ctrl *SolicitudController) GetRegularizacionModal(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(404, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRegularize {
		c.String(403, "No tiene permisos para regularizar fechas")
		return
	}

	utils.Render(c, "solicitud/components/modal_regularizacion", gin.H{
		"Solicitud": solicitud,
	})
}

func (ctrl *SolicitudController) UpdateRegularizacionDates(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(404, "Solicitud no encontrada")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRegularize {
		c.String(403, "No tiene permisos para realizar esta acción")
		return
	}

	fecha := c.PostForm("fecha_regularizacion")

	if err := ctrl.service.UpdateSolicitudDates(c.Request.Context(), id, fecha, fecha); err != nil {
		c.String(400, err.Error())
		return
	}

	// Trigger full page refresh to show updated dates
	c.Header("HX-Refresh", "true")
	c.Status(200)
}

func (ctrl *SolicitudController) Delete(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.JSON(401, gin.H{"error": "No autorizado"})
		return
	}

	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Solicitud no encontrada"})
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanDelete {
		c.JSON(403, gin.H{"error": "No tiene permisos para eliminar esta solicitud"})
		return
	}

	if err := ctrl.service.Delete(c.Request.Context(), id, authUser.ID); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Smart Redirect: If deleting from Show page, redirect to list. If from Table, just trigger reload.
	currentURL := c.GetHeader("HX-Current-URL")
	if strings.Contains(currentURL, "/detalle") {
		// Determine which list to return to (default to derecho)
		redirectURL := "/solicitudes/derecho"
		if strings.Contains(currentURL, "/oficial/") {
			redirectURL = "/solicitudes/oficial"
		}
		c.Header("HX-Location", redirectURL)
	} else {
		c.Header("HX-Trigger", "reloadTable, refresh-stats")
	}

	c.Status(200)
}

func (ctrl *SolicitudController) Finalize(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.JSON(401, gin.H{"error": "No autorizado"})
		return
	}

	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Solicitud no encontrada"})
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanFinalize {
		c.JSON(403, gin.H{"error": "No tiene permisos para finalizar esta solicitud"})
		return
	}

	if err := ctrl.service.Finalize(c.Request.Context(), id); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(200)
}

func (ctrl *SolicitudController) RevertFinalize(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.JSON(401, gin.H{"error": "No autorizado"})
		return
	}

	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"error": "Solicitud no encontrada"})
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRevertFinalize {
		c.JSON(403, gin.H{"error": "No tiene permisos para revertir la finalización"})
		return
	}

	if err := ctrl.service.RevertFinalize(c.Request.Context(), id); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(200)
}

func (ctrl *SolicitudController) IndexOpenTicketDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.service.GetConDescargoOpenTicketPaginated(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)

	// Stats for sidebar/topbar
	pendingRequests, _ := ctrl.service.GetPendingCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())
	pendingDescargos, _ := ctrl.service.GetPendientesDescargo(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())
	var userIDs []string
	if !authUser.IsAdminOrResponsable() {
		userIDs = []string{authUser.ID}
		if senators, err := ctrl.userService.GetSenatorsByEncargado(c.Request.Context(), authUser.ID); err == nil {
			for _, s := range senators {
				userIDs = append(userIDs, s.ID)
			}
		}
	}
	openTicketCount := ctrl.openTicketService.GetPendingCount(c.Request.Context(), userIDs)
	openTicketDescargoCount := ctrl.service.GetConDescargoOpenTicketCount(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable())

	usuariosMap := ctrl.getUsuariosMapFromSolicitudes(c.Request.Context(), result.Solicitudes)

	utils.Render(c, "solicitud/open_ticket_descargo_index", gin.H{
		"Title":                   "Solicitudes con Descargo Open Ticket",
		"Result":                  result,
		"Usuarios":                usuariosMap,
		"LinkBase":                "/solicitudes/con-open-ticket",
		"PendingRequests":         pendingRequests,
		"PendingDescargos":        len(pendingDescargos),
		"OpenTicketCount":         openTicketCount,
		"OpenTicketDescargoCount": openTicketDescargoCount,
		"TotalPending":            pendingRequests + int64(len(pendingDescargos)) + openTicketCount + openTicketDescargoCount,
	})
}

func (ctrl *SolicitudController) TableOpenTicketDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Status(401)
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.service.GetConDescargoOpenTicketPaginated(c.Request.Context(), authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)

	usuariosMap := ctrl.getUsuariosMapFromSolicitudes(c.Request.Context(), result.Solicitudes)

	utils.Render(c, "solicitud/table_solicitudes", gin.H{
		"Result":   result,
		"Usuarios": usuariosMap,
		"LinkBase": "/solicitudes/con-open-ticket",
	})
}

func (ctrl *SolicitudController) getUsuariosMapFromSolicitudes(ctx context.Context, solicitudes []models.Solicitud) map[string]*models.Usuario {
	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}
	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}
	usuariosMap := make(map[string]*models.Usuario)
	if len(ids) > 0 {
		usuarios, _ := ctrl.userService.GetByIDs(ctx, ids)
		for i := range usuarios {
			usuariosMap[usuarios[i].ID] = &usuarios[i]
		}
	}
	return usuariosMap
}
