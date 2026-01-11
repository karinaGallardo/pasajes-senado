package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type AgenciaController struct {
	service *services.AgenciaService
}

func NewAgenciaController() *AgenciaController {
	return &AgenciaController{
		service: services.NewAgenciaService(),
	}
}

func (ctrl *AgenciaController) Index(c *gin.Context) {
	ctx := c.Request.Context()
	agencias, err := ctrl.service.GetAll(ctx)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cargar agencias")
	}

	utils.Render(c, "admin/agencias/index", gin.H{
		"Agencias": agencias,
		"Title":    "Gestión de Agencias",
	})
}

func (ctrl *AgenciaController) New(c *gin.Context) {
	utils.Render(c, "admin/agencias/form_modal", gin.H{
		"Title": "Nueva Agencia",
	})
}

func (ctrl *AgenciaController) Store(c *gin.Context) {
	nombre := c.PostForm("nombre")
	telefono := c.PostForm("telefono")
	estado := c.PostForm("estado") == "on"

	if nombre == "" {
		utils.SetErrorMessage(c, "El nombre es requerido")
		c.Redirect(http.StatusFound, "/admin/agencias")
		return
	}

	_, err := ctrl.service.Create(c.Request.Context(), nombre, telefono, estado)
	if err != nil {
		utils.SetErrorMessage(c, "Error al crear agencia: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Agencia creada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/agencias")
}

func (ctrl *AgenciaController) Edit(c *gin.Context) {
	id := c.Param("id")
	agencia, err := ctrl.service.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Agencia no encontrada")
		return
	}

	utils.Render(c, "admin/agencias/form_modal", gin.H{
		"Agencia": agencia,
		"Title":   "Editar Agencia",
	})
}

func (ctrl *AgenciaController) Update(c *gin.Context) {
	id := c.Param("id")
	nombre := c.PostForm("nombre")
	telefono := c.PostForm("telefono")
	estado := c.PostForm("estado") == "on"

	if nombre == "" {
		utils.SetErrorMessage(c, "El nombre es requerido")
		c.Redirect(http.StatusFound, "/admin/agencias")
		return
	}

	err := ctrl.service.Update(c.Request.Context(), id, nombre, telefono, estado)
	if err != nil {
		utils.SetErrorMessage(c, "Error al actualizar agencia")
	} else {
		utils.SetSuccessMessage(c, "Agencia actualizada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/agencias")
}

func (ctrl *AgenciaController) Toggle(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.service.Toggle(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cambiar estado")
	} else {
		utils.SetSuccessMessage(c, "Estado actualizado correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/agencias")
}

func (ctrl *AgenciaController) Delete(c *gin.Context) {
	id := c.Param("id")
	err := ctrl.service.Delete(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Error al eliminar agencia")
	} else {
		utils.SetSuccessMessage(c, "Agencia eliminada correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/agencias")
}
