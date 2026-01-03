package controllers

import (
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
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
	cats, _ := ctrl.service.GetAllCategorias()
	c.HTML(http.StatusOK, "admin/categorias_compensacion.html", gin.H{
		"Categorias": cats,
		"User":       c.MustGet("User"),
	})
}

func (ctrl *CategoriaCompensacionController) Store(c *gin.Context) {
	dep := c.PostForm("departamento")
	tipo := c.PostForm("tipo_senador")
	monto, _ := strconv.ParseFloat(c.PostForm("monto"), 64)

	if dep != "" && tipo != "" && monto > 0 {
		existing, _ := ctrl.service.FindCategoriaByDepartamentoAndTipo(dep, tipo)
		if existing.ID != "" {
			existing.Monto = monto
			ctrl.service.SaveCategoria(existing)
		} else {
			cat := models.CategoriaCompensacion{
				Departamento: dep,
				TipoSenador:  tipo,
				Monto:        monto,
			}
			err := ctrl.service.SaveCategoria(&cat)
			if err != nil {
				// log.Printf("Error create cat comp: %v", err)
			}
		}
	}
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}

func (ctrl *CategoriaCompensacionController) Delete(c *gin.Context) {
	id := c.Param("id")
	ctrl.service.DeleteCategoria(id)
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}
