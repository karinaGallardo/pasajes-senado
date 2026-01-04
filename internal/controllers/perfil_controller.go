package controllers

import (
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type PerfilController struct {
	ciudadService *services.CiudadService
}

func NewPerfilController() *PerfilController {
	return &PerfilController{
		ciudadService: services.NewCiudadService(),
	}
}

func (ctrl *PerfilController) Show(c *gin.Context) {
	destinos, _ := ctrl.ciudadService.GetAll()

	utils.Render(c, "auth/profile.html", gin.H{
		"Title":    "Mi Perfil",
		"Destinos": destinos,
		"Success":  c.Query("success"),
	})
}
