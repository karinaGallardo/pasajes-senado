package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type CatalogoController struct {
	service *services.CatalogoService
}

func NewCatalogoController() *CatalogoController {
	return &CatalogoController{
		service: services.NewCatalogoService(),
	}
}

func (ctrl *CatalogoController) GetTipos(c *gin.Context) {
	conceptoID := c.Query("concepto_id")
	tipos, _ := ctrl.service.GetTiposByConcepto(conceptoID)

	c.HTML(http.StatusOK, "catalogos/options_tipos.html", gin.H{
		"Tipos": tipos,
	})
}

func (ctrl *CatalogoController) GetAmbitos(c *gin.Context) {
	tipoID := c.Query("tipo_solicitud_id")
	ambitos, _ := ctrl.service.GetAmbitosByTipo(tipoID)

	c.HTML(http.StatusOK, "catalogos/options_ambitos.html", gin.H{
		"Ambitos": ambitos,
	})
}
