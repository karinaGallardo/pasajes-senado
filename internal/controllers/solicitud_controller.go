package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"

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
	ctrl.renderIndex(c, "", "Bandeja de Solicitudes", "solicitud/index")
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

	utils.Render(c, "solicitud/pendientes_descargo", gin.H{
		"Title":              "Solicitudes Pendientes de Descargo",
		"Result":             result,
		"Usuarios":           usuariosMap,
		"PendientesDescargo": true,
		"LinkBase":           "/solicitudes/pendientes-descargo",
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
	ctrl.renderIndex(c, "DERECHO", "Bandeja de Pasajes por Derecho", "solicitud/index")
}

func (ctrl *SolicitudController) IndexOficial(c *gin.Context) {
	ctrl.renderIndex(c, "OFICIAL", "Bandeja de Comisiones Oficiales", "solicitud/index")
}

func (ctrl *SolicitudController) TableDerecho(c *gin.Context) {
	ctrl.renderIndex(c, "DERECHO", "Bandeja de Pasajes por Derecho", "solicitud/table_solicitudes")
}

func (ctrl *SolicitudController) TableOficial(c *gin.Context) {
	ctrl.renderIndex(c, "OFICIAL", "Bandeja de Comisiones Oficiales", "solicitud/table_solicitudes")
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

	linkBase := "/solicitudes"
	switch concepto {
	case "DERECHO":
		linkBase = "/solicitudes/derecho"
	case "OFICIAL":
		linkBase = "/solicitudes/oficial"
	}

	utils.Render(c, template, gin.H{
		"Title":                title,
		"Result":               result,
		"Usuarios":             usuariosMap,
		"Status":               status,
		"Concepto":             concepto,
		"LinkBase":             linkBase,
		"IsAdminOrResponsable": authUser.IsAdminOrResponsable(),
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

	utils.Render(c, "layouts/components/pending_stats", gin.H{
		"PendingRequests":  pendingRequests,
		"PendingDescargos": len(pendingDescargos),
		"TotalPending":     pendingRequests + int64(len(pendingDescargos)),
	})
}

func (ctrl *SolicitudController) GetRegularizacionModal(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(404, "Solicitud no encontrada")
		return
	}

	utils.Render(c, "solicitud/components/modal_regularizacion", gin.H{
		"Solicitud": solicitud,
	})
}

func (ctrl *SolicitudController) UpdateRegularizacionDates(c *gin.Context) {
	id := c.Param("id")
	fecha := c.PostForm("fecha_regularizacion")

	if err := ctrl.service.UpdateSolicitudDates(c.Request.Context(), id, fecha, fecha); err != nil {
		c.String(400, err.Error())
		return
	}

	// Trigger full page refresh to show updated dates
	c.Header("HX-Refresh", "true")
	c.Status(200)
}
