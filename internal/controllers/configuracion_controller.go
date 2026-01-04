package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type ConfiguracionController struct {
	service *services.ConfiguracionService
}

func NewConfiguracionController() *ConfiguracionController {
	return &ConfiguracionController{
		service: services.NewConfiguracionService(),
	}
}

func (ctrl *ConfiguracionController) Index(c *gin.Context) {
	configs, _ := ctrl.service.GetAll()

	utils.Render(c, "admin/configuracion.html", gin.H{
		"Configs": configs,
		"Title":   "Configuración del Sistema",
	})
}

func (ctrl *ConfiguracionController) Update(c *gin.Context) {
	clave := c.PostForm("clave")
	valor := c.PostForm("valor")

	if clave != "" && valor != "" {
		ctrl.service.Update(clave, valor)
	}
	c.Redirect(http.StatusFound, "/admin/configuracion")
}
