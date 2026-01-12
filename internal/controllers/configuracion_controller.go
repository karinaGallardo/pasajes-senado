package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
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
	configs, _ := ctrl.service.GetAll(c.Request.Context())

	utils.Render(c, "admin/configuracion", gin.H{
		"Configs": configs,
		"Title":   "Configuración del Sistema",
	})
}

func (ctrl *ConfiguracionController) Update(c *gin.Context) {
	var req dtos.UpdateConfiguracionRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/configuracion")
		return
	}

	conf := models.Configuracion{
		Clave: req.Clave,
		Valor: req.Valor,
	}

	ctrl.service.Update(c.Request.Context(), &conf)
	c.Redirect(http.StatusFound, "/admin/configuracion")
}
