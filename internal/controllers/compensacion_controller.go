package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type CompensacionController struct {
	compService *services.CompensacionService
	userService *services.UsuarioService
}

func NewCompensacionController() *CompensacionController {
	return &CompensacionController{
		compService: services.NewCompensacionService(),
		userService: services.NewUsuarioService(),
	}
}

func (ctrl *CompensacionController) Index(c *gin.Context) {
	list, _ := ctrl.compService.GetAll(c.Request.Context())
	utils.Render(c, "compensacion/index", gin.H{
		"Title": "Gestión de Compensaciones",
		"Lista": list,
	})
}

func (ctrl *CompensacionController) Create(c *gin.Context) {
	users, _ := ctrl.userService.GetAll(c.Request.Context())
	cats, _ := ctrl.compService.GetAllCategorias(c.Request.Context())

	utils.Render(c, "compensacion/create", gin.H{
		"Title":      "Nueva Compensación",
		"Usuarios":   users,
		"Categorias": cats,
	})
}

func (ctrl *CompensacionController) Store(c *gin.Context) {
	var req dtos.CreateCompensacionRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	fechaInicio, _ := time.Parse("2006-01-02", req.FechaInicio)
	fechaFin, _ := time.Parse("2006-01-02", req.FechaFin)
	total, _ := strconv.ParseFloat(req.Total, 64)
	retencion, _ := strconv.ParseFloat(req.Retencion, 64)

	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeSuffix, _ := gonanoid.Generate(alphabet, 6)
	codigo := fmt.Sprintf("COMP-%d-%s", time.Now().Year(), codeSuffix)

	comp := models.Compensacion{
		Codigo:          codigo,
		Nombre:          req.NombreTramite,
		FuncionarioID:   req.FuncionarioID,
		FechaInicio:     fechaInicio,
		FechaFin:        fechaFin,
		MesCompensacion: req.Mes,
		Estado:          "BORRADOR",
		Glosa:           req.Glosa,
		Total:           total,
		Retencion:       retencion,
		Informe:         req.Informe,
	}

	if err := ctrl.compService.Create(c.Request.Context(), &comp); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/compensaciones")
}
