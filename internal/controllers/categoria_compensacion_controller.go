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

type CategoriaCompensacionController struct {
	service *services.CompensacionService
}

func NewCategoriaCompensacionController() *CategoriaCompensacionController {
	return &CategoriaCompensacionController{
		service: services.NewCompensacionService(),
	}
}

func (ctrl *CategoriaCompensacionController) Index(c *gin.Context) {
	cats, _ := ctrl.service.GetAllCategorias(c.Request.Context())
	utils.Render(c, "admin/categorias_compensacion", gin.H{
		"Categorias": cats,
	})
}

func (ctrl *CategoriaCompensacionController) Store(c *gin.Context) {
	var req dtos.CreateCategoriaCompensacionRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
		return
	}

	dep := req.Departamento
	tipo := req.TipoSenador
	monto, _ := strconv.ParseFloat(req.Monto, 64)

	if dep != "" && tipo != "" && monto > 0 {
		existing, _ := ctrl.service.GetCategoriaByDepartamentoAndTipo(c.Request.Context(), dep, tipo)
		if existing.ID != "" {
			existing.Monto = monto
			ctrl.service.UpdateCategoria(c.Request.Context(), existing)
		} else {
			cat := models.CategoriaCompensacion{
				Departamento: dep,
				TipoSenador:  tipo,
				Monto:        monto,
			}
			err := ctrl.service.CreateCategoria(c.Request.Context(), &cat)
			if err != nil {
				// log.Printf("Error create cat comp: %v", err)
			}
		}
	}
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}

func (ctrl *CategoriaCompensacionController) Delete(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteCategoria(c.Request.Context(), id)
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}
