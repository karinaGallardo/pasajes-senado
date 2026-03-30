package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type ReportController struct {
	reportService    *services.ReportService
	aerolineaService *services.AerolineaService
	agenciaService   *services.AgenciaService
}

func NewReportController(reportService *services.ReportService, aerolineaService *services.AerolineaService, agenciaService *services.AgenciaService) *ReportController {
	return &ReportController{
		reportService:    reportService,
		aerolineaService: aerolineaService,
		agenciaService:   agenciaService,
	}
}

func (ctrl *ReportController) Index(c *gin.Context) {
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())

	utils.Render(c, "admin/reports/index", gin.H{
		"Title":      "Reportes del Sistema",
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
	})
}

func (ctrl *ReportController) DownloadConsolidadoExcel(c *gin.Context) {
	var filter dtos.ReportFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		utils.SetErrorMessage(c, "Filtros inválidos")
		c.Redirect(http.StatusFound, "/admin/reports")
		return
	}

	f, err := ctrl.reportService.GenerateConsolidadoPasajesExcel(c.Request.Context(), filter)
	if err != nil {
		utils.SetErrorMessage(c, "Error generando reporte: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/reports")
		return
	}

	fileName := fmt.Sprintf("Reporte_Pasajes_%s.xlsx", utils.FormatDateFilename())
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)

	if err := f.Write(c.Writer); err != nil {
		// Too late to redirect if headers sent
	}
}

func (ctrl *ReportController) DownloadMorosidadExcel(c *gin.Context) {
	f, err := ctrl.reportService.GenerateMorosidadDescargosExcel(c.Request.Context())
	if err != nil {
		utils.SetErrorMessage(c, "Error generando reporte: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/reports")
		return
	}

	fileName := fmt.Sprintf("Reporte_Morosidad_%s.xlsx", utils.FormatDateFilename())
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	_ = f.Write(c.Writer)
}

func (ctrl *ReportController) DownloadUsoCuposExcel(c *gin.Context) {
	anio := utils.StrToInt(c.Query("anio"), 2026)
	mes := utils.StrToInt(c.Query("mes"), 3)

	f, err := ctrl.reportService.GenerateUsoCuposExcel(c.Request.Context(), anio, mes)
	if err != nil {
		utils.SetErrorMessage(c, "Error generando reporte: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/reports")
		return
	}

	fileName := fmt.Sprintf("Reporte_Cupos_%d_%d.xlsx", anio, mes)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	_ = f.Write(c.Writer)
}

func (ctrl *ReportController) DownloadEstadisticasAerolineaExcel(c *gin.Context) {
	var filter dtos.ReportFilterRequest
	_ = c.ShouldBindQuery(&filter)

	f, err := ctrl.reportService.GenerateEstadisticasAerolineaExcel(c.Request.Context(), filter)
	if err != nil {
		utils.SetErrorMessage(c, "Error generando reporte: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/reports")
		return
	}

	fileName := fmt.Sprintf("Estadisticas_Aerolineas_%s.xlsx", utils.FormatDateFilename())
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	_ = f.Write(c.Writer)
}
