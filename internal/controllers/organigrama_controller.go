package controllers

import (
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
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
	cargos, _ := ctrl.service.GetAllCargos()
	c.HTML(http.StatusOK, "admin/cargos.html", gin.H{
		"Cargos": cargos,
		"User":   c.MustGet("User"),
	})
}

func (ctrl *OrganigramaController) StoreCargo(c *gin.Context) {
	codigo, _ := strconv.Atoi(c.PostForm("codigo"))
	descripcion := c.PostForm("descripcion")
	categoria, _ := strconv.Atoi(c.PostForm("categoria"))

	cargo := models.Cargo{
		Codigo:      codigo,
		Descripcion: descripcion,
		Categoria:   categoria,
	}

	if err := ctrl.service.CreateCargo(&cargo); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) DeleteCargo(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteCargo(id)
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) IndexOficinas(c *gin.Context) {
	oficinas, _ := ctrl.service.GetAllOficinas()
	c.HTML(http.StatusOK, "admin/oficinas.html", gin.H{
		"Oficinas": oficinas,
		"User":     c.MustGet("User"),
	})
}

func (ctrl *OrganigramaController) StoreOficina(c *gin.Context) {
	codigo, _ := strconv.Atoi(c.PostForm("codigo"))
	detalle := c.PostForm("detalle")
	area := c.PostForm("area")
	presupuesto, _ := strconv.ParseFloat(c.PostForm("presupuesto"), 64)

	oficina := models.Oficina{
		Codigo:      codigo,
		Detalle:     detalle,
		Area:        area,
		Presupuesto: presupuesto,
	}

	if err := ctrl.service.CreateOficina(&oficina); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/oficinas")
}

func (ctrl *OrganigramaController) DeleteOficina(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteOficina(id)
	c.Redirect(http.StatusFound, "/admin/oficinas")
}
