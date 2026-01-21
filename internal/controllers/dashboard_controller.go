package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"time"

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
	user := appcontext.CurrentUser(c)
	var senadoresCalculados []models.Usuario

	if user != nil {
		assigned, err := ctrl.usuarioService.GetSenatorsByEncargado(c.Request.Context(), user.ID)
		if err == nil && len(assigned) > 0 {
			senadoresCalculados = assigned
		}
	}

	var solicitudes []models.Solicitud
	if user != nil && user.IsAdminOrResponsable() {
		solicitudes, _ = ctrl.solicitudService.GetAll(c.Request.Context())
	} else if user != nil {
		solicitudes, _ = ctrl.solicitudService.GetByUserID(c.Request.Context(), user.ID)
	}

	descargos, _ := ctrl.descargoService.GetAll(c.Request.Context())

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

	now := time.Now()
	utils.Render(c, "dashboard/index", gin.H{
		"Title":               "Panel de Control",
		"Pendientes":          pendientes,
		"Aprobados":           aprobados,
		"Descargos":           len(descargos),
		"Recent":              solicitudes,
		"SenadoresEncargados": senadoresCalculados,
		"Gestion":             now.Year(),
		"Mes":                 int(now.Month()),
	})
}
