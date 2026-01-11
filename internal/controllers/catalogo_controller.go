package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type CatalogoController struct {
	tipoSolicitudService *services.TipoSolicitudService
}

func NewCatalogoController() *CatalogoController {
	return &CatalogoController{
		tipoSolicitudService: services.NewTipoSolicitudService(),
	}
}

func (ctrl *CatalogoController) GetTipos(c *gin.Context) {
	conceptoID := c.Query("concepto_id")
	tipos, _ := ctrl.tipoSolicitudService.GetByConcepto(c.Request.Context(), conceptoID)

	c.HTML(http.StatusOK, "catalogos/options_tipos", gin.H{
		"Tipos": tipos,
	})
}

func (ctrl *CatalogoController) GetAmbitos(c *gin.Context) {
	tipoID := c.Query("tipo_solicitud_id")
	ambitos, _ := ctrl.tipoSolicitudService.GetAmbitosByTipo(c.Request.Context(), tipoID)

	c.HTML(http.StatusOK, "catalogos/options_ambitos", gin.H{
		"Ambitos": ambitos,
	})
}
