package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	service     *services.SolicitudService
	userService *services.UsuarioService
}

func NewSolicitudController(service *services.SolicitudService, userService *services.UsuarioService) *SolicitudController {
	return &SolicitudController{
		service:     service,
		userService: userService,
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

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPendientesDescargoPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		// handle empty
	}

	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)
	scopedUserIDs := ctrl.userService.GetScopedUserIDs(ctx, authUser)
	stats := ctrl.service.GetPendingStats(ctx, authUser.ID, authUser.IsAdminOrResponsable(), scopedUserIDs)

	utils.Render(c, "solicitud/pendientes_descargo", gin.H{
		"Title":                   "Solicitudes Pendientes de Descargo",
		"Result":                  result,
		"Usuarios":                usuariosMap,
		"PendientesDescargo":      true,
		"LinkBase":                "/solicitudes/pendientes-descargo",
		"PendingRequests":         stats.PendingRequests,
		"PendingDescargos":        stats.PendingDescargos,
		"OpenTicketCount":         stats.OpenTicketCount,
		"OpenTicketDescargoCount": stats.OpenTicketDescargoCount,
		"EnRevisionCount":         stats.EnRevisionCount,
		"TotalPending":            stats.TotalPending,
	})
}

func (ctrl *SolicitudController) TablePendientesDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPendientesDescargoPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		// handle empty
	}

	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)

	utils.Render(c, "solicitud/table_pendientes_descargo", gin.H{
		"Result":             result,
		"Usuarios":           usuariosMap,
		"PendientesDescargo": true,
		"LinkBase":           "/solicitudes/pendientes-descargo",
	})
}

func (ctrl *SolicitudController) EnRevisionDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetEnRevisionDescargoPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		utils.Render(c, "error.html", gin.H{"error": err.Error()})
		return
	}

	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)
	scopedUserIDs := ctrl.userService.GetScopedUserIDs(ctx, authUser)
	stats := ctrl.service.GetPendingStats(ctx, authUser.ID, authUser.IsAdminOrResponsable(), scopedUserIDs)

	utils.Render(c, "solicitud/en_revision_descargo", gin.H{
		"Title":                   "Descargos en Revisión",
		"Result":                  result,
		"Usuarios":                usuariosMap,
		"EnRevisionDescargo":      true,
		"LinkBase":                "/solicitudes/en-revision-descargo",
		"PendingRequests":         stats.PendingRequests,
		"PendingDescargos":        stats.PendingDescargos,
		"OpenTicketCount":         stats.OpenTicketCount,
		"OpenTicketDescargoCount": stats.OpenTicketDescargoCount,
		"EnRevisionCount":         stats.EnRevisionCount,
		"TotalPending":            stats.TotalPending,
	})
}

func (ctrl *SolicitudController) TableEnRevisionDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetEnRevisionDescargoPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	if err != nil {
		if c.GetHeader("HX-Request") == "true" {
			c.String(500, "Error al cargar la tabla: "+err.Error())
		} else {
			utils.Render(c, "error.html", gin.H{"error": err.Error()})
		}
		return
	}

	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)

	utils.Render(c, "solicitud/table_en_revision_descargo", gin.H{
		"Result":             result,
		"Usuarios":           usuariosMap,
		"EnRevisionDescargo": true,
		"LinkBase":           "/solicitudes/en-revision-descargo",
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

	ctx := c.Request.Context()
	status := c.Query("estado")
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := ctrl.service.GetPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), status, concepto, page, limit, searchTerm)
	if err != nil {
		// handle or initialize empty result
	}

	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)

	for i := range result.Solicitudes {
		result.Solicitudes[i].HydratePermissions(authUser)
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

	ctx := c.Request.Context()
	scopedUserIDs := ctrl.userService.GetScopedUserIDs(ctx, authUser)
	stats := ctrl.service.GetPendingStats(ctx, authUser.ID, authUser.IsAdminOrResponsable(), scopedUserIDs)

	utils.Render(c, "layouts/components/pending_stats", gin.H{
		"PendingRequests":         stats.PendingRequests,
		"PendingDescargos":        stats.PendingDescargos,
		"OpenTicketCount":         stats.OpenTicketCount,
		"OpenTicketDescargoCount": stats.OpenTicketDescargoCount,
		"EnRevisionCount":         stats.EnRevisionCount,
		"TotalPending":            stats.TotalPending,
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

	if err := ctrl.service.Delete(c.Request.Context(), id, authUser); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	currentURL := c.GetHeader("HX-Current-URL")
	if strings.Contains(currentURL, "/detalle") {
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

	if err := ctrl.service.Finalize(c.Request.Context(), id, authUser); err != nil {
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

	if err := ctrl.service.RevertFinalize(c.Request.Context(), id, authUser); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(200)
}

func (ctrl *SolicitudController) RevertReject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.JSON(401, gin.H{"error": "No autorizado"})
		return
	}

	if err := ctrl.service.RevertReject(c.Request.Context(), id, authUser); err != nil {
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

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.service.GetConDescargoOpenTicketPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)

	scopedUserIDs := ctrl.userService.GetScopedUserIDs(ctx, authUser)
	stats := ctrl.service.GetPendingStats(ctx, authUser.ID, authUser.IsAdminOrResponsable(), scopedUserIDs)
	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)

	utils.Render(c, "solicitud/open_ticket_descargo_index", gin.H{
		"Title":                   "Solicitudes con Descargo Open Ticket",
		"Result":                  result,
		"Usuarios":                usuariosMap,
		"LinkBase":                "/solicitudes/con-open-ticket",
		"PendingRequests":         stats.PendingRequests,
		"PendingDescargos":        stats.PendingDescargos,
		"OpenTicketCount":         stats.OpenTicketCount,
		"OpenTicketDescargoCount": stats.OpenTicketDescargoCount,
		"TotalPending":            stats.TotalPending,
	})
}

func (ctrl *SolicitudController) TableOpenTicketDescargo(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Status(401)
		return
	}

	ctx := c.Request.Context()
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.service.GetConDescargoOpenTicketPaginated(ctx, authUser.ID, authUser.IsAdminOrResponsable(), page, limit, searchTerm)
	usuariosMap := ctrl.userService.BuildUsuariosMapFromSolicitudes(ctx, result.Solicitudes)

	utils.Render(c, "solicitud/table_solicitudes", gin.H{
		"Result":   result,
		"Usuarios": usuariosMap,
		"LinkBase": "/solicitudes/con-open-ticket",
	})
}
