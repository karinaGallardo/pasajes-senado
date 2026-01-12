package utils

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

// Render procesa plantillas HTML inyectando tokens CSRF, contexto del usuario, roles y mensajes flash.
func Render(c *gin.Context, templateName string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	data["csrf_token"] = csrf.GetToken(c)

	user := appcontext.CurrentUser(c)
	if user != nil {
		data["CurrentUser"] = user
		role := ""
		if user.Rol != nil {
			role = user.Rol.Codigo
		}
		data["IsAdmin"] = role == "ADMIN"
		data["IsResponsable"] = role == "RESPONSABLE"
		data["IsUsuario"] = role == "USUARIO"
		data["IsSenador"] = role == "SENADOR"
		data["IsFuncionario"] = role == "FUNCIONARIO"

		data["CanManageSystem"] = role == "ADMIN" || role == "RESPONSABLE"
		data["CanManageUsers"] = role == "ADMIN"
	}

	session := sessions.Default(c)
	success := session.Flashes("success")
	if len(success) > 0 {
		data["SuccessMessage"] = success[0]
	}

	errors := session.Flashes("error")
	if len(errors) > 0 {
		data["ErrorMessage"] = errors[0]
	}
	session.Save()

	c.HTML(http.StatusOK, templateName, data)
}

// SetSuccessMessage registra un mensaje flash de éxito en la sesión actual.
func SetSuccessMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "success")
	session.Save()
}

// SetErrorMessage registra un mensaje flash de error en la sesión actual.
func SetErrorMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "error")
	session.Save()
}
