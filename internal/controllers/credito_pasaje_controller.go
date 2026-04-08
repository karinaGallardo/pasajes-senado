package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type CreditoPasajeController struct {
	service *services.CreditoPasajeService
}

func NewCreditoPasajeController(service *services.CreditoPasajeService) *CreditoPasajeController {
	return &CreditoPasajeController{service: service}
}

func (ctrl *CreditoPasajeController) ListByUser(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	creditos, err := ctrl.service.GetByUsuarioID(c.Request.Context(), user.ID)
	if err != nil {
		utils.SetErrorMessage(c, "Error al obtener créditos de pasaje")
		c.Redirect(302, "/dashboard")
		return
	}

	utils.Render(c, "perfil/creditos.html", gin.H{
		"Title":    "Mis Créditos de Viaje",
		"Creditos": creditos,
	})
}
