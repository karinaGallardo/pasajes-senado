package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UsuarioController struct {
	userService   *services.UsuarioService
	rolService    *services.RolService
	ciudadService *services.CiudadService
}

func NewUsuarioController() *UsuarioController {
	return &UsuarioController{
		userService:   services.NewUsuarioService(),
		rolService:    services.NewRolService(),
		ciudadService: services.NewCiudadService(),
	}
}

func (ctrl *UsuarioController) Index(c *gin.Context) {
	roleType := c.DefaultQuery("rol", "SENADOR")
	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))

	msg := c.Query("msg")

	var result interface{}
	var err error

	if roleType == "FUNCIONARIO" {
		result, err = ctrl.userService.GetPaginated(roleType, page, limit, searchTerm)
	} else {
		usuarios, _ := ctrl.userService.GetByRoleType(roleType)
		result = gin.H{"Usuarios": usuarios}
	}

	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/index.html", gin.H{
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
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))

	var result interface{}
	var err error

	if roleType == "FUNCIONARIO" {
		result, err = ctrl.userService.GetPaginated(roleType, page, limit, searchTerm)
	} else {
		usuarios, _ := ctrl.userService.GetByRoleType(roleType)
		result = gin.H{"Usuarios": usuarios}
	}

	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/table", gin.H{
		"Result": result,
		"Rol":    roleType,
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

	utils.Render(c, "usuarios/edit_modal", gin.H{
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

	rolCodigo := c.PostForm("rol_codigo")
	if rolCodigo != "" {
		usuario.RolCodigo = &rolCodigo
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

func (ctrl *UsuarioController) UpdateOrigin(c *gin.Context) {
	targetID := c.Param("id")
	origenCode := c.PostForm("origen_code")

	if origenCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Debe seleccionar una ciudad"})
		return
	}

	targetUser, err := ctrl.userService.GetByID(targetID)
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
	isAdmin := currentUser.Rol != nil && currentUser.Rol.Codigo == "ADMIN"
	isSelf := targetUser.ID == currentUser.ID

	if !isEncargado && !isAdmin && !isSelf {
		c.JSON(http.StatusForbidden, gin.H{"error": "No tiene permisos para modificar este usuario"})
		return
	}

	targetUser.OrigenCode = &origenCode
	if err := ctrl.userService.Update(targetUser); err != nil {
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
		count, err = ctrl.userService.SyncSenators()
	} else {
		count, err = ctrl.userService.SyncStaff()
	}

	if err != nil {
		c.String(http.StatusInternalServerError, "Error sincronizando: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/usuarios?rol="+roleType+"&msg=Sincronizados "+strconv.Itoa(count)+" registros")
}
