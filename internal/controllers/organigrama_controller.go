package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrganigramaController struct {
	service *services.OrganigramaService
}

func NewOrganigramaController() *OrganigramaController {
	return &OrganigramaController{
		service: services.NewOrganigramaService(),
	}
}

func (ctrl *OrganigramaController) IndexCargos(c *gin.Context) {
	cargos, _ := ctrl.service.GetAllCargos(c.Request.Context())
	utils.Render(c, "admin/cargos", gin.H{
		"Cargos": cargos,
	})
}

func (ctrl *OrganigramaController) StoreCargo(c *gin.Context) {
	var req dtos.CreateCargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cargos")
		return
	}

	codigo, _ := strconv.Atoi(req.Codigo)
	categoria, _ := strconv.Atoi(req.Categoria)

	cargo := models.Cargo{
		Codigo:      codigo,
		Descripcion: req.Descripcion,
		Categoria:   categoria,
	}

	if err := ctrl.service.CreateCargo(c.Request.Context(), &cargo); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) DeleteCargo(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteCargo(c.Request.Context(), id)
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) IndexOficinas(c *gin.Context) {
	oficinas, _ := ctrl.service.GetAllOficinas(c.Request.Context())
	utils.Render(c, "admin/oficinas", gin.H{
		"Oficinas": oficinas,
	})
}

func (ctrl *OrganigramaController) StoreOficina(c *gin.Context) {
	var req dtos.CreateOficinaRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/oficinas")
		return
	}

	codigo, _ := strconv.Atoi(req.Codigo)
	presupuesto, _ := strconv.ParseFloat(req.Presupuesto, 64)

	oficina := models.Oficina{
		Codigo:      codigo,
		Detalle:     req.Detalle,
		Area:        req.Area,
		Presupuesto: presupuesto,
	}

	if err := ctrl.service.CreateOficina(c.Request.Context(), &oficina); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/oficinas")
}

func (ctrl *OrganigramaController) DeleteOficina(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteOficina(c.Request.Context(), id)
	c.Redirect(http.StatusFound, "/admin/oficinas")
}
