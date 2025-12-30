package controllers

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"

	"github.com/gin-gonic/gin"
)

type OrganigramaController struct {
	cargoRepo   *repositories.CargoRepository
	oficinaRepo *repositories.OficinaRepository
}

func NewOrganigramaController() *OrganigramaController {
	db := configs.DB
	return &OrganigramaController{
		cargoRepo:   repositories.NewCargoRepository(db),
		oficinaRepo: repositories.NewOficinaRepository(db),
	}
}

func (ctrl *OrganigramaController) IndexCargos(c *gin.Context) {
	cargos, _ := ctrl.cargoRepo.FindAll()
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

	if err := ctrl.cargoRepo.Create(&cargo); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) DeleteCargo(c *gin.Context) {
	id := c.Param("id")
	ctrl.cargoRepo.Delete(id)
	c.Redirect(http.StatusFound, "/admin/cargos")
}

func (ctrl *OrganigramaController) IndexOficinas(c *gin.Context) {
	oficinas, _ := ctrl.oficinaRepo.FindAll()
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

	if err := ctrl.oficinaRepo.Create(&oficina); err != nil {
	}
	c.Redirect(http.StatusFound, "/admin/oficinas")
}

func (ctrl *OrganigramaController) DeleteOficina(c *gin.Context) {
	id := c.Param("id")
	ctrl.oficinaRepo.Delete(id)
	c.Redirect(http.StatusFound, "/admin/oficinas")
}
