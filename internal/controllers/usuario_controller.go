package controllers

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService   *services.UsuarioService
	rolService    *services.RolService
	ciudadService *services.CiudadService
}

func NewUsuarioController() *UsuarioController {
	db := configs.DB
	return &UsuarioController{
		userService:   services.NewUsuarioService(db),
		rolService:    services.NewRolService(db),
		ciudadService: services.NewCiudadService(db),
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
	destinos, _ := ctrl.ciudadService.GetAll()

	funcionarios, _ := ctrl.userService.GetByRoleType("FUNCIONARIO")

	c.HTML(http.StatusOK, "usuarios/edit_modal", gin.H{
		"Usuario":      usuario,
		"Roles":        roles,
		"Destinos":     destinos,
		"Funcionarios": funcionarios,
	})
}

func (ctrl *UsuarioController) Update(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	rolID := c.PostForm("rol_id")
	if rolID != "" {
		usuario.RolID = &rolID
	}

	origen := c.PostForm("origen")
	if origen != "" {
		usuario.OrigenCode = &origen
	} else {
		usuario.OrigenCode = nil
	}

	encargadoID := c.PostForm("encargado_id")
	if encargadoID != "" {
		usuario.EncargadoID = &encargadoID
	} else {
		usuario.EncargadoID = nil
	}

	if err := ctrl.userService.Update(usuario); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando usuario")
		return
	}

	c.Header("HX-Trigger", "reloadTable")
	c.Status(http.StatusOK)
}
