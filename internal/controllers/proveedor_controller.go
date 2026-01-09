package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type ProveedorController struct {
	aerolineaService *services.AerolineaService
	agenciaService   *services.AgenciaService
}

func NewProveedorController() *ProveedorController {
	return &ProveedorController{
		aerolineaService: services.NewAerolineaService(),
		agenciaService:   services.NewAgenciaService(),
	}
}

func (ctrl *ProveedorController) Index(c *gin.Context) {
	aerolineas, _ := ctrl.aerolineaService.GetAll(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAll(c.Request.Context())

	utils.Render(c, "admin/proveedores.html", gin.H{
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
		"Title":      "Gestión de Proveedores",
	})
}

func (ctrl *ProveedorController) CreateAerolinea(c *gin.Context) {
	nombre := c.PostForm("nombre")
	if nombre != "" {
		ctrl.aerolineaService.Create(c.Request.Context(), nombre)
	}
	c.Redirect(http.StatusFound, "/admin/proveedores")
}

func (ctrl *ProveedorController) ToggleAerolinea(c *gin.Context) {
	id := c.Param("id")
	ctrl.aerolineaService.Toggle(c.Request.Context(), id)
	c.Redirect(http.StatusFound, "/admin/proveedores")
}

func (ctrl *ProveedorController) CreateAgencia(c *gin.Context) {
	nombre := c.PostForm("nombre")
	telefono := c.PostForm("telefono")
	if nombre != "" {
		ctrl.agenciaService.Create(c.Request.Context(), nombre, telefono)
	}
	c.Redirect(http.StatusFound, "/admin/proveedores")
}

func (ctrl *ProveedorController) ToggleAgencia(c *gin.Context) {
	id := c.Param("id")
	ctrl.agenciaService.Toggle(c.Request.Context(), id)
	c.Redirect(http.StatusFound, "/admin/proveedores")
}
