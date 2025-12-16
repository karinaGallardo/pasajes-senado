package controllers

import (
	"net/http"
	"sistema-pasajes/internal/repositories"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	solicitudRepo *repositories.SolicitudRepository
	descargoRepo  *repositories.DescargoRepository
}

func NewDashboardController() *DashboardController {
	return &DashboardController{
		solicitudRepo: repositories.NewSolicitudRepository(),
		descargoRepo:  repositories.NewDescargoRepository(),
	}
}

func (ctrl *DashboardController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.solicitudRepo.FindAll()
	descargos, _ := ctrl.descargoRepo.FindAll()

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
