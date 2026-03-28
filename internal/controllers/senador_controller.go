package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SenadorController struct {
	userService  *services.UsuarioService
	auditService *services.AuditService
}

func NewSenadorController(
	userService *services.UsuarioService,
	auditService *services.AuditService,
) *SenadorController {
	return &SenadorController{
		userService:  userService,
		auditService: auditService,
	}
}

func (ctrl *SenadorController) Index(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	msg := c.Query("msg")

	usuarios, err := ctrl.userService.GetByRoleType(c.Request.Context(), "SENADOR")
	if err != nil {
		// log.Printf("Error: %v", err)
	}

	result := gin.H{
		"Usuarios":    usuarios,
		"CurrentYear": time.Now().Year(),
		"AuthUser":    authUser,
	}

	utils.Render(c, "usuarios/senadores", gin.H{
		"Title":      "Senadores",
		"Result":     result,
		"Rol":        "SENADOR",
		"Msg":        msg,
		"SearchTerm": searchTerm,
	})
}

func (ctrl *SenadorController) Table(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Status(http.StatusUnauthorized)
		return
	}

	searchTerm := c.Query("q")

	usuarios, err := ctrl.userService.GetByRoleType(c.Request.Context(), "SENADOR")
	if err != nil {
		// log.Printf("Error: %v", err)
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

	result := gin.H{
		"Usuarios":    usuarios,
		"CurrentYear": time.Now().Year(),
		"AuthUser":    authUser,
	}

	utils.Render(c, "usuarios/table_senadores", result)
}

func (ctrl *SenadorController) Sync(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No autorizado")
		return
	}

	result, err := ctrl.userService.SyncSenators(c.Request.Context())
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

	c.Redirect(http.StatusFound, "/usuarios/senadores")
}

func (ctrl *SenadorController) GetSyncModal(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No autorizado")
		return
	}
	utils.Render(c, "usuarios/components/modal_sync_confirm", gin.H{
		"Rol": "SENADOR",
	})
}
