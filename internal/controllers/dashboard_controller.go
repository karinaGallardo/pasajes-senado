package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

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
	solicitudes, _ := ctrl.solicitudService.FindAll(c.Request.Context())
	descargos, _ := ctrl.descargoService.FindAll(c.Request.Context())

	user := appcontext.CurrentUser(c)
	var senadoresCalculados []models.Usuario

	if user != nil {
		assigned, err := ctrl.usuarioService.GetSenatorsByEncargado(c.Request.Context(), user.ID)
		if err == nil && len(assigned) > 0 {
			senadoresCalculados = assigned
		}
	}

	var pendientes, aprobados, finalizados int
	for _, s := range solicitudes {
		st := "SOLICITADO"
		if s.EstadoSolicitudCodigo != nil {
			st = *s.EstadoSolicitudCodigo
		}
		switch st {
		case "SOLICITADO":
			pendientes++
		case "APROBADO":
			aprobados++
		case "FINALIZADO":
			finalizados++
		}
	}

	utils.Render(c, "dashboard/index", gin.H{
		"Title":               "Panel de Control",
		"Pendientes":          pendientes,
		"Aprobados":           aprobados,
		"Descargos":           len(descargos),
		"Recent":              solicitudes,
		"SenadoresEncargados": senadoresCalculados,
	})
}
