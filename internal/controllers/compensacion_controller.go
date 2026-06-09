package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type CompensacionController struct {
	compService *services.CompensacionService
	userService *services.UsuarioService
}

func NewCompensacionController(compService *services.CompensacionService, userService *services.UsuarioService) *CompensacionController {
	return &CompensacionController{
		compService: compService,
		userService: userService,
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

	_, err := ctrl.compService.CreateFromRequest(c.Request.Context(), req)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/compensaciones")
}
