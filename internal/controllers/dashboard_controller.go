package controllers

import (
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	solicitudService *services.SolicitudService
	descargoService  *services.DescargoService
	usuarioService   *services.UsuarioService
}

func NewDashboardController() *DashboardController {
	return &DashboardController{
		solicitudService: services.NewSolicitudService(),
		descargoService:  services.NewDescargoService(),
		usuarioService:   services.NewUsuarioService(),
	}
}

func (ctrl *DashboardController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.solicitudService.FindAll()
	descargos, _ := ctrl.descargoService.FindAll()

	userVal, exists := c.Get("User")
	var senadoresCalculados []models.Usuario

	if exists {
		if u, ok := userVal.(*models.Usuario); ok {
			assigned, err := ctrl.usuarioService.GetSenatorsByEncargado(u.ID)
			if err == nil && len(assigned) > 0 {
				senadoresCalculados = assigned
			}
		}
	}

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
		"Title":               "Panel de Control",
		"User":                c.MustGet("User"),
		"Pendientes":          pendientes,
		"Aprobados":           aprobados,
		"Descargos":           len(descargos),
		"Recent":              solicitudes,
		"SenadoresEncargados": senadoresCalculados,
	})
}
