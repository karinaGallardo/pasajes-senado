package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type FuncionarioController struct {
	userService  *services.UsuarioService
	auditService *services.AuditService
}

func NewFuncionarioController(
	userService *services.UsuarioService,
	auditService *services.AuditService,
) *FuncionarioController {
	return &FuncionarioController{
		userService:  userService,
		auditService: auditService,
	}
}

func (ctrl *FuncionarioController) Index(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	msg := c.Query("msg")

	result, err := ctrl.userService.GetPaginated(c.Request.Context(), "FUNCIONARIO", page, limit, searchTerm)
	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/funcionarios", gin.H{
		"Title":      "Funcionarios",
		"Result":     result,
		"Rol":        "FUNCIONARIO",
		"Msg":        msg,
		"SearchTerm": searchTerm,
	})
}

func (ctrl *FuncionarioController) Table(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	result, err := ctrl.userService.GetPaginated(c.Request.Context(), "FUNCIONARIO", page, limit, searchTerm)
	if err != nil {
		// log.Printf("Error: %v", err)
	}

	utils.Render(c, "usuarios/table_funcionarios", gin.H{"Result": result})
}

func (ctrl *FuncionarioController) Sync(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No autorizado")
		return
	}

	result, err := ctrl.userService.SyncStaff(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Error sincronizando: "+err.Error())
		return
	}

	utils.SetSuccessMessage(c, "Sincronizados "+strconv.Itoa(result.Count)+" registros")

	if len(result.Conflicts) > 0 {
		var msg strings.Builder
		msg.WriteString("Se detectaron conflictos de unicidad: ")
		for i, cnf := range result.Conflicts {
			if i > 0 {
				msg.WriteString(" | ")
			}
			msg.WriteString(cnf)
		}
		utils.SetErrorMessage(c, msg.String())
	}

	c.Redirect(http.StatusFound, "/usuarios/funcionarios")
}

func (ctrl *FuncionarioController) GetSyncModal(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No autorizado")
		return
	}
	utils.Render(c, "usuarios/components/modal_sync_confirm", gin.H{
		"Rol": "FUNCIONARIO",
	})
}
