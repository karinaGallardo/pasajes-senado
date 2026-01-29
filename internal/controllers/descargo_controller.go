package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type DescargoController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
}

func NewDescargoController() *DescargoController {
	return &DescargoController{
		descargoService:  services.NewDescargoService(),
		solicitudService: services.NewSolicitudService(),
	}
}

func (ctrl *DescargoController) Index(c *gin.Context) {
	descargos, _ := ctrl.descargoService.GetAll(c.Request.Context())
	utils.Render(c, "descargo/index", gin.H{
		"Title":     "Bandeja de Descargos (FV-05)",
		"Descargos": descargos,
	})
}

func (ctrl *DescargoController) Create(c *gin.Context) {
	solicitudID := c.Query("solicitud_id")
	if solicitudID == "" {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	existe, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/"+existe.ID)
		return
	}

	utils.Render(c, "descargo/create", gin.H{
		"Title":     "Nuevo Descargo",
		"Solicitud": solicitud,
	})
}

func (ctrl *DescargoController) Store(c *gin.Context) {
	var req dtos.CreateDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes?error=DatosInvalidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if _, err := ctrl.descargoService.Create(c.Request.Context(), req, authUser.ID); err != nil {
		log.Printf("Error creando descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos")
}

func (ctrl *DescargoController) Show(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error buscando descargo %s: %v", id, err)
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	utils.Render(c, "descargo/show", gin.H{
		"Title":    "Detalle de Descargo",
		"Descargo": descargo,
	})
}

func (ctrl *DescargoController) Approve(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error aprobando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/"+id)
		return
	}

	descargo.Estado = "APROBADO"
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}
	descargo.UpdatedBy = &authUser.ID

	ctrl.descargoService.Update(c.Request.Context(), descargo)

	if descargo.SolicitudID != "" {
		ctrl.solicitudService.Finalize(c.Request.Context(), descargo.SolicitudID)
	}

	c.Redirect(http.StatusFound, "/descargos/"+id)
}
