package controllers

import (
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type CategoriaViaticoController struct {
	service     *services.CategoriaViaticoService
	viaticoServ *services.ViaticoService
}

func NewCategoriaViaticoController() *CategoriaViaticoController {
	return &CategoriaViaticoController{
		service:     services.NewCategoriaViaticoService(),
		viaticoServ: services.NewViaticoService(),
	}
}

func (ctrl *CategoriaViaticoController) Index(c *gin.Context) {
	categorias, err := ctrl.service.GetAll(c.Request.Context())
	if err != nil {
		c.String(500, "Error listando categorías: "+err.Error())
		return
	}

	zonas, _ := ctrl.viaticoServ.GetZonas(c.Request.Context())

	utils.Render(c, "admin/viatico/categorias", gin.H{
		"Title":      "Categorías de Viáticos",
		"Categorias": categorias,
		"Zonas":      zonas,
	})
}

func (ctrl *CategoriaViaticoController) Store(c *gin.Context) {
	var req dtos.CreateCategoriaViaticoRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos: "+err.Error())
		c.Redirect(303, "/admin/viaticos/categorias")
		return
	}

	cat := &models.CategoriaViatico{
		Nombre:        req.Nombre,
		Codigo:        req.Codigo,
		Monto:         req.Monto,
		Moneda:        req.Moneda,
		ZonaViaticoID: &req.ZonaViaticoID,
	}

	if err := ctrl.service.Create(c.Request.Context(), cat); err != nil {
		utils.SetErrorMessage(c, "Error al crear categoría: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Categoría creada con éxito")
	}

	c.Redirect(303, "/admin/viaticos/categorias")
}

func (ctrl *CategoriaViaticoController) StoreZona(c *gin.Context) {
	var req dtos.CreateZonaViaticoRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos: "+err.Error())
		c.Redirect(303, "/admin/viaticos/categorias")
		return
	}

	if err := ctrl.viaticoServ.CreateZona(c.Request.Context(), req.Nombre); err != nil {
		utils.SetErrorMessage(c, "Error al crear zona: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Zona creada con éxito")
	}

	c.Redirect(303, "/admin/viaticos/categorias")
}
