package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type AerolineaController struct {
	service *services.AerolineaService
}

func NewAerolineaController() *AerolineaController {
	return &AerolineaController{
		service: services.NewAerolineaService(),
	}
}

func (ctrl *AerolineaController) Index(c *gin.Context) {
	ctx := c.Request.Context()
	aerolineas, err := ctrl.service.GetAll(ctx)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cargar aerolíneas")
	}

	utils.Render(c, "admin/aerolineas/index", gin.H{
		"Aerolineas": aerolineas,
		"Title":      "Gestión de Aerolíneas",
	})
}

func (ctrl *AerolineaController) New(c *gin.Context) {
	utils.Render(c, "admin/aerolineas/form_modal", gin.H{
		"Title": "Nueva Aerolínea",
	})
}

func (ctrl *AerolineaController) Store(c *gin.Context) {
	var req dtos.CreateAerolineaRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/aerolineas")
		return
	}

	model := models.Aerolinea{
		Nombre: req.Nombre,
		Estado: req.Estado == "on",
	}

	if err := ctrl.service.Create(c.Request.Context(), &model); err != nil {
		utils.SetErrorMessage(c, "Error al crear aerolínea: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Aerolínea creada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/aerolineas")
}

func (ctrl *AerolineaController) Edit(c *gin.Context) {
	id := c.Param("id")
	aerolinea, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Aerolínea no encontrada")
		return
	}

	utils.Render(c, "admin/aerolineas/form_modal", gin.H{
		"Aerolinea": aerolinea,
		"Title":     "Editar Aerolínea",
	})
}

func (ctrl *AerolineaController) Update(c *gin.Context) {
	id := c.Param("id")
	var req dtos.UpdateAerolineaRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Redirect(http.StatusFound, "/admin/aerolineas")
		return
	}

	existing, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Aerolínea no encontrada")
		c.Redirect(http.StatusFound, "/admin/aerolineas")
		return
	}

	existing.Nombre = req.Nombre
	existing.Estado = req.Estado == "on"

	if err := ctrl.service.Update(c.Request.Context(), existing); err != nil {
		utils.SetErrorMessage(c, "Error al actualizar aerolínea")
	} else {
		utils.SetSuccessMessage(c, "Aerolínea actualizada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/aerolineas")
}

func (ctrl *AerolineaController) Toggle(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.service.Toggle(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cambiar estado")
	} else {
		utils.SetSuccessMessage(c, "Estado actualizado correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/aerolineas")
}

func (ctrl *AerolineaController) Delete(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.service.Delete(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Error al eliminar aerolínea")
	} else {
		utils.SetSuccessMessage(c, "Aerolínea eliminada correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/aerolineas")
}
