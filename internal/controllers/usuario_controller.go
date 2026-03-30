package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"log/slog"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService  *services.UsuarioService
	auditService *services.AuditService
}

func NewUsuarioController(
	userService *services.UsuarioService,
	auditService *services.AuditService,
) *UsuarioController {
	return &UsuarioController{
		userService:  userService,
		auditService: auditService,
	}
}

func (ctrl *UsuarioController) Edit(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	ctx, err := ctrl.userService.GetEditContext(c.Request.Context(), id, authUser)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	data := gin.H{
		"Usuario":      ctx.Usuario,
		"Roles":        ctx.Roles,
		"Destinos":     ctx.Destinos,
		"Funcionarios": ctx.Funcionarios,
		"Cargos":       ctx.Cargos,
		"Oficinas":     ctx.Oficinas,
	}
	for k, v := range ctx.Permissions {
		data[k] = v
	}

	viewName := "usuarios/edit"
	if c.GetHeader("HX-Request") == "true" {
		viewName = "usuarios/components/usuario_edit_modal"
	}

	utils.Render(c, viewName, data)
}

func (ctrl *UsuarioController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	ctx, err := ctrl.userService.GetEditContext(c.Request.Context(), id, authUser)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	data := gin.H{
		"Usuario":      ctx.Usuario,
		"Roles":        ctx.Roles,
		"Destinos":     ctx.Destinos,
		"Funcionarios": ctx.Funcionarios,
		"Cargos":       ctx.Cargos,
		"Oficinas":     ctx.Oficinas,
	}
	for k, v := range ctx.Permissions {
		data[k] = v
	}

	utils.Render(c, "usuarios/components/usuario_edit_modal", data)
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

	if authUser.IsAdminOrResponsable() {
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
	}

	usuario.Email = req.Email
	usuario.Phone = req.Phone

	var successMsg, errorMsg string

	if err := ctrl.userService.Update(c.Request.Context(), usuario); err != nil {
		slog.Error("Error actualizando usuario", "id", id, "err", err)
		errorMsg = "Error al actualizar datos básicos: " + err.Error()
	} else {
		// Sincronizar Orígenes Alternativos si es privilegiado y es senador
		if authUser.IsAdminOrResponsable() && usuario.IsSenador() {
			if err := ctrl.userService.SyncOrigenesAlternativos(c.Request.Context(), usuario.ID, req.OrigenesAlternativos); err != nil {
				slog.Error("Error sincronizando orígenes alternativos", "id", usuario.ID, "err", err)
				errorMsg = "Error al sincronizar orígenes alternativos: " + err.Error()
			}
		}
	}

	if errorMsg == "" {
		successMsg = "Cambios guardados exitosamente"
		// AUDITORÍA: Registrar el cambio
		go ctrl.auditService.Log(c.Request.Context(), "UPDATE_USER", "Usuario", id, "", "Actualización de perfil/logística", "", "")
	}

	editCtx, _ := ctrl.userService.GetEditContext(c.Request.Context(), id, authUser)
	data := gin.H{
		"Usuario":      editCtx.Usuario,
		"Roles":        editCtx.Roles,
		"Destinos":     editCtx.Destinos,
		"Funcionarios": editCtx.Funcionarios,
		"Cargos":       editCtx.Cargos,
		"Oficinas":     editCtx.Oficinas,
		"Success":      successMsg,
		"Error":        errorMsg,
	}
	for k, v := range editCtx.Permissions {
		data[k] = v
	}

	if c.GetHeader("HX-Request") == "true" {
		if errorMsg == "" {
			c.Header("HX-Trigger", "reloadTable")
		}
		utils.Render(c, "usuarios/components/usuario_edit_modal", data)
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

	isEncargado := targetUser.IsManagedBy(authUser)
	isPrivileged := authUser.IsAdminOrResponsable()
	isSelf := authUser.IsOwner(targetID)

	if !isEncargado && !isPrivileged && !isSelf {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tiene permisos para modificar este usuario"})
		return
	}

	targetUser.OrigenIATA = &req.OrigenCode
	if err := ctrl.userService.Update(c.Request.Context(), targetUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar: " + err.Error()})
		return
	}

	// AUDITORÍA: Registrar el cambio de origen
	go ctrl.auditService.Log(c.Request.Context(), "UPDATE_USER_ORIGIN", "Usuario", targetID, "", "Actualización manual de origen", "", "")

	c.JSON(http.StatusOK, gin.H{"message": "Origen actualizado correctamente"})
}

func (ctrl *UsuarioController) Unblock(c *gin.Context) {
	id := c.Param("id")
	usuario, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Usuario no encontrado"})
		return
	}

	usuario.Unblock()

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
