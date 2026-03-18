package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService        *services.UsuarioService
	rolService         *services.RolService
	destinoService     *services.DestinoService
	organigramaService *services.OrganigramaService
}

func NewUsuarioController(
	userService *services.UsuarioService,
	rolService *services.RolService,
	destinoService *services.DestinoService,
	organigramaService *services.OrganigramaService,
) *UsuarioController {
	return &UsuarioController{
		userService:        userService,
		rolService:         rolService,
		destinoService:     destinoService,
		organigramaService: organigramaService,
	}
}

func (ctrl *UsuarioController) Edit(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}
	roles, _ := ctrl.rolService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	funcionarios, _ := ctrl.userService.GetByRoleType(c.Request.Context(), "FUNCIONARIO")
	cargos, _ := ctrl.organigramaService.GetAllCargos(c.Request.Context())
	oficinas, _ := ctrl.organigramaService.GetAllOficinas(c.Request.Context())

	viewName := "usuarios/edit"
	if c.GetHeader("HX-Request") == "true" {
		viewName = "usuarios/components/edit_modal"
	}

	utils.Render(c, viewName, gin.H{
		"Usuario":      usuario,
		"Roles":        roles,
		"Destinos":     destinos,
		"Funcionarios": funcionarios,
		"Cargos":       cargos,
		"Oficinas":     oficinas,
	})
}

func (ctrl *UsuarioController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}
	roles, _ := ctrl.rolService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	funcionarios, _ := ctrl.userService.GetByRoleType(c.Request.Context(), "FUNCIONARIO")
	cargos, _ := ctrl.organigramaService.GetAllCargos(c.Request.Context())
	oficinas, _ := ctrl.organigramaService.GetAllOficinas(c.Request.Context())

	utils.Render(c, "usuarios/components/edit_modal", gin.H{
		"Usuario":      usuario,
		"Roles":        roles,
		"Destinos":     destinos,
		"Funcionarios": funcionarios,
		"Cargos":       cargos,
		"Oficinas":     oficinas,
	})
}

func (ctrl *UsuarioController) Update(c *gin.Context) {
	id := c.Param("id")

	var req dtos.UpdateUsuarioRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos")
		return
	}

	usuario, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.String(http.StatusUnauthorized, "No autorizado")
		return
	}

	isPrivileged := false

	if authUser.IsAdminOrResponsable() {
		isPrivileged = true
	} else if authUser.ID == usuario.ID {
		isPrivileged = true
	} else if usuario.EncargadoID != nil && *usuario.EncargadoID == authUser.ID {
		isPrivileged = true
	}

	if !isPrivileged {
		c.String(http.StatusForbidden, "No tiene permisos para modificar este usuario")
		return
	}

	if req.RolCodigo != "" {
		usuario.RolCodigo = &req.RolCodigo
	}

	if req.OrigenIATA != "" {
		usuario.OrigenIATA = &req.OrigenIATA
	} else {
		usuario.OrigenIATA = nil
	}

	if req.EncargadoID != "" {
		usuario.EncargadoID = &req.EncargadoID
	} else {
		usuario.EncargadoID = nil
	}

	usuario.Email = req.Email
	usuario.Phone = req.Phone

	if err := ctrl.userService.Update(c.Request.Context(), usuario); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando usuario")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		referer := c.Request.Header.Get("Referer")
		if strings.Contains(referer, "/perfil") || strings.Contains(referer, "/cupos/derecho") {
			c.Header("HX-Refresh", "true")
		} else {
			c.Header("HX-Trigger", "reloadTable")
		}
		c.Status(http.StatusOK)
		return
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}

func (ctrl *UsuarioController) UpdateOrigin(c *gin.Context) {
	targetID := c.Param("id")

	var req dtos.UpdateUserOriginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe seleccionar una ciudad"})
		return
	}

	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autorizado"})
		return
	}

	isEncargado := targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID
	isPrivileged := authUser.IsAdminOrResponsable()
	isSelf := targetUser.ID == authUser.ID

	if !isEncargado && !isPrivileged && !isSelf {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tiene permisos para modificar este usuario"})
		return
	}

	targetUser.OrigenIATA = &req.OrigenCode
	if err := ctrl.userService.Update(c.Request.Context(), targetUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar: " + err.Error()})
		return
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}

func (ctrl *UsuarioController) Unblock(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	usuario.IsBlocked = false
	usuario.LoginAttempts = 0

	if err := ctrl.userService.Update(c.Request.Context(), usuario); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al desbloquear usuario"})
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Trigger", "reloadTable")
		c.Status(http.StatusOK)
		return
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/dashboard"
	}
	c.Redirect(http.StatusFound, referer)
}
