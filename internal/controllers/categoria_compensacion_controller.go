package controllers

import (
	"net/http"
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
	utils.Render(c, "admin/categorias_compensacion.html", gin.H{
		"Categorias": cats,
	})
}

func (ctrl *CategoriaCompensacionController) Store(c *gin.Context) {
	dep := c.PostForm("departamento")
	tipo := c.PostForm("tipo_senador")
	monto, _ := strconv.ParseFloat(c.PostForm("monto"), 64)

	if dep != "" && tipo != "" && monto > 0 {
		existing, _ := ctrl.service.FindCategoriaByDepartamentoAndTipo(c.Request.Context(), dep, tipo)
		if existing.ID != "" {
			existing.Monto = monto
			ctrl.service.SaveCategoria(c.Request.Context(), existing)
		} else {
			cat := models.CategoriaCompensacion{
				Departamento: dep,
				TipoSenador:  tipo,
				Monto:        monto,
			}
			err := ctrl.service.SaveCategoria(c.Request.Context(), &cat)
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
