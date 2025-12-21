package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService *services.UsuarioService
	rolService  *services.RolService
}

func NewUsuarioController() *UsuarioController {
	return &UsuarioController{
		userService: services.NewUsuarioService(),
		rolService:  services.NewRolService(),
	}
}

func (ctrl *UsuarioController) Index(c *gin.Context) {
	usuarios, _ := ctrl.userService.GetByRoleType("SENADOR")

	c.HTML(http.StatusOK, "usuarios/index.html", gin.H{
		"Title":    "Gestión de Usuarios",
		"User":     c.MustGet("User"),
		"Usuarios": usuarios,
		"Type":     "SENADOR",
	})
}

func (ctrl *UsuarioController) Table(c *gin.Context) {
	roleType := c.DefaultQuery("type", "SENADOR")
	usuarios, _ := ctrl.userService.GetByRoleType(roleType)

	c.HTML(http.StatusOK, "usuarios/table", gin.H{
		"Usuarios": usuarios,
	})
}

func (ctrl *UsuarioController) Edit(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}
	roles, _ := ctrl.rolService.GetAll()

	c.HTML(http.StatusOK, "usuarios/edit_modal", gin.H{
		"Usuario": usuario,
		"Roles":   roles,
	})
}

func (ctrl *UsuarioController) Update(c *gin.Context) {
	id := c.Param("id")
	rolCodigo := c.PostForm("rol_id")

	if err := ctrl.userService.UpdateRol(id, rolCodigo); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando rol")
		return
	}

	c.Header("HX-Trigger", "reloadTable")
	c.Status(http.StatusOK)
}
