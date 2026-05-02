package controllers

import (
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AuditController struct {
	auditService *services.AuditService
}

func NewAuditController(auditService *services.AuditService) *AuditController {
	return &AuditController{auditService: auditService}
}

func (ctrl *AuditController) Index(c *gin.Context) {
	filters := map[string]string{
		"action":      c.Query("action"),
		"entity_type": c.Query("entity_type"),
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	logs, total, err := ctrl.auditService.GetAll(c.Request.Context(), filters, limit, offset)
	if err != nil {
		c.String(500, "Error obteniendo logs: %v", err)
		return
	}

	actions, entities, _ := ctrl.auditService.GetAvailableFilters(c.Request.Context())

	utils.Render(c, "admin/audit/index", gin.H{
		"Title":             "Registro de Auditoría",
		"Logs":              logs,
		"Total":             total,
		"AvailableActions":  actions,
		"AvailableEntities": entities,
		"CurrentFilters":    filters,
		"Page":              page,
		"Limit":             limit,
		"TotalPages":        (int(total) + limit - 1) / limit,
	})
}

func (ctrl *AuditController) Table(c *gin.Context) {
	filters := map[string]string{
		"action":      c.Query("action"),
		"entity_type": c.Query("entity_type"),
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	logs, total, err := ctrl.auditService.GetAll(c.Request.Context(), filters, limit, offset)
	if err != nil {
		c.String(500, "Error obteniendo logs: %v", err)
		return
	}

	utils.Render(c, "admin/audit/table", gin.H{
		"Logs":           logs,
		"Page":           page,
		"Limit":          limit,
		"Total":          total,
		"TotalPages":     (int(total) + limit - 1) / limit,
		"CurrentFilters": filters,
	})
}
