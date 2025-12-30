package controllers

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type PerfilController struct {
	ciudadService *services.CiudadService
}

func NewPerfilController() *PerfilController {
	db := configs.DB
	return &PerfilController{
		ciudadService: services.NewCiudadService(db),
	}
}

func (ctrl *PerfilController) Show(c *gin.Context) {
	userContext, _ := c.Get("User")
	user := userContext.(*models.Usuario)

	destinos, _ := ctrl.ciudadService.GetAll()

	c.HTML(http.StatusOK, "auth/profile.html", gin.H{
		"Title":    "Mi Perfil",
		"User":     user,
		"Destinos": destinos,
		"Success":  c.Query("success"),
	})
}
