package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CategoriaCompensacionController struct {
	repo *repositories.CategoriaCompensacionRepository
}

func NewCategoriaCompensacionController() *CategoriaCompensacionController {
	db := configs.DB
	return &CategoriaCompensacionController{
		repo: repositories.NewCategoriaCompensacionRepository(db),
	}
}

func (ctrl *CategoriaCompensacionController) Index(c *gin.Context) {
	cats, _ := ctrl.repo.FindAll()
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
		existing, _ := ctrl.repo.FindByDepartamentoAndTipo(dep, tipo)
		if existing.ID != "" {
			existing.Monto = monto
			ctrl.repo.Save(existing)
		} else {
			cat := models.CategoriaCompensacion{
				Departamento: dep,
				TipoSenador:  tipo,
				Monto:        monto,
			}
			err := ctrl.repo.Save(&cat)
			if err != nil {
				fmt.Printf("Error create cat comp: %v", err)
			}
		}
	}
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}

func (ctrl *CategoriaCompensacionController) Delete(c *gin.Context) {
	id := c.Param("id")
	ctrl.repo.Delete(id)
	c.Redirect(http.StatusFound, "/admin/compensaciones/categorias")
}
