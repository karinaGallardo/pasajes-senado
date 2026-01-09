package controllers

import (
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type PerfilController struct {
	destinoService *services.DestinoService
}

func NewPerfilController() *PerfilController {
	return &PerfilController{
		destinoService: services.NewDestinoService(),
	}
}

func (ctrl *PerfilController) Show(c *gin.Context) {
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	utils.Render(c, "auth/profile.html", gin.H{
		"Title":    "Mi Perfil",
		"Destinos": destinos,
		"Success":  c.Query("success"),
	})
}
