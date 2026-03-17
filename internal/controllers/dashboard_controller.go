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

func NewDashboardController(solicitudService *services.SolicitudService, descargoService *services.DescargoService, usuarioService *services.UsuarioService) *DashboardController {
	return &DashboardController{
		solicitudService: solicitudService,
		descargoService:  descargoService,
		usuarioService:   usuarioService,
	}
}

func (ctrl *DashboardController) Index(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	// --- Construir el scope de usuarios según el rol ---
	//
	// ADMIN / RESPONSABLE_PASAJES → ve todo (sin filtro)
	// ENCARGADO_PASAJES           → ve sus propias solicitudes + las de sus senadores asignados
	// SENADOR / FUNCIONARIO       → solo sus propias solicitudes

	isAdminOrResp := authUser.IsAdminOrResponsable()

	// Senadores que atiende este usuario (encargado)
	var senadoresCalculados []models.Usuario
	if assigned, err := ctrl.usuarioService.GetSenatorsByEncargado(c.Request.Context(), authUser.ID); err == nil && len(assigned) > 0 {
		senadoresCalculados = assigned
	}

	// Obtener solicitudes según el scope
	var solicitudes []models.Solicitud
	if isAdminOrResp {
		solicitudes, _ = ctrl.solicitudService.GetAll(c.Request.Context(), "", "")
	} else {
		solicitudes, _ = ctrl.solicitudService.GetByUserIdOrAccesibleByEncargadoID(c.Request.Context(), authUser.ID, "", "")
	}

	// Construir lista de user IDs para el conteo de descargos
	// Para admins: nil → GetAll; para el resto: los IDs del scope
	var descargoCount int64
	if isAdminOrResp {
		descargos, _ := ctrl.descargoService.GetAll(c.Request.Context())
		descargoCount = int64(len(descargos))
	} else {
		// IDs en scope: el propio usuario + sus senadores asignados
		scopeIDs := []string{authUser.ID}
		for _, s := range senadoresCalculados {
			scopeIDs = append(scopeIDs, s.ID)
		}
		descargoCount = ctrl.descargoService.GetCountByUserIDs(c.Request.Context(), scopeIDs)
	}

	// Contar estados en las solicitudes del scope
	var pendientes, aprobados int
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
		}
	}

	now := time.Now()
	utils.Render(c, "dashboard/index", gin.H{
		"Title":               "Panel de Control",
		"Pendientes":          pendientes,
		"Aprobados":           aprobados,
		"Descargos":           descargoCount,
		"Recent":              solicitudes,
		"SenadoresEncargados": senadoresCalculados,
		"Gestion":             now.Year(),
		"Mes":                 int(now.Month()),
	})
}
