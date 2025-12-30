package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type CompensacionController struct {
	repo     *repositories.CompensacionRepository
	userRepo *repositories.UsuarioRepository
	catRepo  *repositories.CategoriaCompensacionRepository
}

func NewCompensacionController() *CompensacionController {
	db := configs.DB
	return &CompensacionController{
		repo:     repositories.NewCompensacionRepository(db),
		userRepo: repositories.NewUsuarioRepository(db),
		catRepo:  repositories.NewCategoriaCompensacionRepository(db),
	}
}

func (ctrl *CompensacionController) Index(c *gin.Context) {
	list, _ := ctrl.repo.FindAll()
	c.HTML(http.StatusOK, "compensacion/index.html", gin.H{
		"Title": "Gestión de Compensaciones",
		"Lista": list,
		"User":  c.MustGet("User"),
	})
}

func (ctrl *CompensacionController) Create(c *gin.Context) {
	users, _ := ctrl.userRepo.FindAll()
	cats, _ := ctrl.catRepo.FindAll()

	c.HTML(http.StatusOK, "compensacion/create.html", gin.H{
		"Title":      "Nueva Compensación",
		"Usuarios":   users,
		"Categorias": cats,
		"User":       c.MustGet("User"),
	})
}

func (ctrl *CompensacionController) Store(c *gin.Context) {
	funcionarioID := c.PostForm("funcionario_id")
	fechaInicio, _ := time.Parse("2006-01-02", c.PostForm("fecha_inicio"))
	fechaFin, _ := time.Parse("2006-01-02", c.PostForm("fecha_fin"))
	total, _ := strconv.ParseFloat(c.PostForm("total"), 64)
	retencion, _ := strconv.ParseFloat(c.PostForm("retencion"), 64)

	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeSuffix, _ := gonanoid.Generate(alphabet, 6)
	codigo := fmt.Sprintf("COMP-%d-%s", time.Now().Year(), codeSuffix)

	comp := models.Compensacion{
		Codigo:          codigo,
		Nombre:          c.PostForm("nombre_tramite"),
		FuncionarioID:   funcionarioID,
		FechaInicio:     fechaInicio,
		FechaFin:        fechaFin,
		MesCompensacion: c.PostForm("mes"),
		Estado:          "BORRADOR",
		Glosa:           c.PostForm("glosa"),
		Total:           total,
		Retencion:       retencion,
		Informe:         c.PostForm("informe"),
	}

	if err := ctrl.repo.Create(&comp); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/compensaciones")
}
