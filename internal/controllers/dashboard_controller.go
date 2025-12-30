package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	solicitudService *services.SolicitudService
	descargoService  *services.DescargoService
}

func NewDashboardController() *DashboardController {
	return &DashboardController{
		solicitudService: services.NewSolicitudService(),
		descargoService:  services.NewDescargoService(),
	}
}

func (ctrl *DashboardController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.solicitudService.FindAll()
	descargos, _ := ctrl.descargoService.FindAll()

	var pendientes, aprobados, finalizados int
	for _, s := range solicitudes {
		switch s.Estado {
		case "SOLICITADO":
			pendientes++
		case "APROBADO":
			aprobados++
		case "FINALIZADO":
			finalizados++
		}
	}

	c.HTML(http.StatusOK, "dashboard/index.html", gin.H{
		"Title":      "Panel de Control",
		"User":       c.MustGet("User"),
		"Pendientes": pendientes,
		"Aprobados":  aprobados,
		"Descargos":  len(descargos),
		"Recent":     solicitudes,
	})
}
