package controllers

import (
	"net/http"
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
	nombre := c.PostForm("nombre")
	estado := c.PostForm("estado") == "on"

	if nombre == "" {
		utils.SetErrorMessage(c, "El nombre es requerido")
		c.Redirect(http.StatusFound, "/admin/aerolineas")
		return
	}

	_, err := ctrl.service.Create(c.Request.Context(), nombre, estado)
	if err != nil {
		utils.SetErrorMessage(c, "Error al crear aerolínea: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Aerolínea creada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/aerolineas")
}

func (ctrl *AerolineaController) Edit(c *gin.Context) {
	id := c.Param("id")
	aerolinea, err := ctrl.service.FindByID(c.Request.Context(), id)
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
	nombre := c.PostForm("nombre")
	estado := c.PostForm("estado") == "on"

	if nombre == "" {
		utils.SetErrorMessage(c, "El nombre es requerido")
		c.Redirect(http.StatusFound, "/admin/aerolineas")
		return
	}

	err := ctrl.service.Update(c.Request.Context(), id, nombre, estado)
	if err != nil {
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
