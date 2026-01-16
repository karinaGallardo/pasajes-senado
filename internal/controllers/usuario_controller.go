package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService    *services.UsuarioService
	rolService     *services.RolService
	destinoService *services.DestinoService
}

func NewUsuarioController() *UsuarioController {
	return &UsuarioController{
		userService:    services.NewUsuarioService(),
		rolService:     services.NewRolService(),
		destinoService: services.NewDestinoService(),
	}
}

func (ctrl *UsuarioController) Index(c *gin.Context) {
	roleType := c.DefaultQuery("rol", "SENADOR")
	searchTerm := c.Query("q")
	page := c.GetInt("page")
	limit := c.GetInt("limit")

	msg := c.Query("msg")

	var result any
	var err error

	if roleType == "FUNCIONARIO" {
		result, err = ctrl.userService.GetPaginated(c.Request.Context(), roleType, page, limit, searchTerm)
	} else {
		usuarios, errDb := ctrl.userService.GetByRoleType(c.Request.Context(), roleType)
		if errDb != nil {
			err = errDb
		}
		result = gin.H{"Usuarios": usuarios}
	}

	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/index", gin.H{
		"Title":      "Gestión de Usuarios",
		"Result":     result,
		"Rol":        roleType,
		"Msg":        msg,
		"SearchTerm": searchTerm,
	})
}

func (ctrl *UsuarioController) Table(c *gin.Context) {
	roleType := c.DefaultQuery("rol", "SENADOR")
	searchTerm := c.Query("q")
	page := c.GetInt("page")
	limit := c.GetInt("limit")

	var result interface{}
	var err error

	if roleType == "FUNCIONARIO" {
		result, err = ctrl.userService.GetPaginated(c.Request.Context(), roleType, page, limit, searchTerm)
	} else {
		usuarios, errDb := ctrl.userService.GetByRoleType(c.Request.Context(), roleType)
		if errDb != nil {
			err = errDb
		}

		if searchTerm != "" {
			var filtered []models.Usuario
			lowerTerm := strings.ToLower(searchTerm)
			for _, u := range usuarios {
				match := strings.Contains(strings.ToLower(u.GetNombreCompleto()), lowerTerm) ||
					strings.Contains(strings.ToLower(u.CI), lowerTerm) ||
					strings.Contains(strings.ToLower(u.Username), lowerTerm) ||
					strings.Contains(strings.ToLower(u.Email), lowerTerm)

				if !match {
					suplente := u.GetSuplente()
					if suplente != nil {
						match = strings.Contains(strings.ToLower(suplente.GetNombreCompleto()), lowerTerm) ||
							strings.Contains(strings.ToLower(suplente.CI), lowerTerm) ||
							strings.Contains(strings.ToLower(suplente.Username), lowerTerm) ||
							strings.Contains(strings.ToLower(suplente.Email), lowerTerm)
					}
				}

				if match {
					filtered = append(filtered, u)
				}
			}
			usuarios = filtered
		}

		result = gin.H{"Usuarios": usuarios}
	}

	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/components/table", gin.H{
		"Result": result,
		"Rol":    roleType,
	})
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

	viewName := "usuarios/edit"
	if c.GetHeader("HX-Request") == "true" {
		viewName = "usuarios/components/edit_modal"
	}

	utils.Render(c, viewName, gin.H{
		"Usuario":      usuario,
		"Roles":        roles,
		"Destinos":     destinos,
		"Funcionarios": funcionarios,
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

	utils.Render(c, "usuarios/components/edit_modal", gin.H{
		"Usuario":      usuario,
		"Roles":        roles,
		"Destinos":     destinos,
		"Funcionarios": funcionarios,
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

	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil {
		c.String(http.StatusUnauthorized, "No autorizado")
		return
	}

	isPrivileged := false

	if currentUser.IsAdminOrResponsable() {
		isPrivileged = true
	} else if currentUser.ID == usuario.ID {
		isPrivileged = true
	} else if usuario.EncargadoID != nil && *usuario.EncargadoID == currentUser.ID {
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

	if err := ctrl.userService.Update(c.Request.Context(), usuario); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando usuario")
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		referer := c.Request.Header.Get("Referer")
		if strings.Contains(referer, "/perfil") {
			c.Header("HX-Refresh", "true")
		} else {
			c.Header("HX-Trigger", "reloadTable")
		}
		c.Status(http.StatusOK)
		return
	}

	referer := c.Request.Header.Get("Referer")
	if referer == "" {
		referer = "/usuarios"
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

	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No autorizado"})
		return
	}

	isEncargado := targetUser.EncargadoID != nil && *targetUser.EncargadoID == currentUser.ID
	isAdmin := currentUser.IsAdmin()
	isSelf := targetUser.ID == currentUser.ID

	if !isEncargado && !isAdmin && !isSelf {
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

func (ctrl *UsuarioController) Sync(c *gin.Context) {
	roleType := c.DefaultQuery("rol", "SENADOR")
	var count int
	var err error

	if roleType == "SENADOR" {
		count, err = ctrl.userService.SyncSenators(c.Request.Context())
	} else {
		count, err = ctrl.userService.SyncStaff(c.Request.Context())
	}

	if err != nil {
		c.String(http.StatusInternalServerError, "Error sincronizando: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Sincronizados "+strconv.Itoa(count)+" registros")
	c.Redirect(http.StatusFound, "/usuarios?rol="+roleType)
}
func (ctrl *UsuarioController) GetSyncModal(c *gin.Context) {
	roleType := c.DefaultQuery("rol", "SENADOR")
	utils.Render(c, "usuarios/components/modal_sync_confirm", gin.H{
		"Rol": roleType,
	})
}
