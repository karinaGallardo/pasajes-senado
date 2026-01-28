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
	service      *services.ConfiguracionService
	emailService *services.EmailService
}

func NewConfiguracionController() *ConfiguracionController {
	return &ConfiguracionController{
		service:      services.NewConfiguracionService(),
		emailService: services.NewEmailService(),
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

func (ctrl *ConfiguracionController) TestEmail(c *gin.Context) {
	email := c.PostForm("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	err := ctrl.emailService.SendEmail([]string{email}, "Test Email de Sistema Pasajes", "<h1>Correo de Prueba</h1><p>Si ves esto, la configuración SMTP funciona correctamente.</p>")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email enviado correctamente"})
}
