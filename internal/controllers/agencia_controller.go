package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
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
	var req dtos.CreateAgenciaRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/agencias")
		return
	}

	model := models.Agencia{
		Nombre:   req.Nombre,
		Telefono: req.Telefono,
		Estado:   req.Estado == "on",
	}

	if err := ctrl.service.Create(c.Request.Context(), &model); err != nil {
		utils.SetErrorMessage(c, "Error al crear agencia: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Agencia creada correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/agencias")
}

func (ctrl *AgenciaController) Edit(c *gin.Context) {
	id := c.Param("id")
	agencia, err := ctrl.service.GetByID(c.Request.Context(), id)
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
	var req dtos.UpdateAgenciaRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Redirect(http.StatusFound, "/admin/agencias")
		return
	}

	existing, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Agencia no encontrada")
		c.Redirect(http.StatusFound, "/admin/agencias")
		return
	}

	existing.Nombre = req.Nombre
	existing.Telefono = req.Telefono
	existing.Estado = req.Estado == "on"

	if err := ctrl.service.Update(c.Request.Context(), existing); err != nil {
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
