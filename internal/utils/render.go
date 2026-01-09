package utils

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	csrf "github.com/utrack/gin-csrf"
)

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
		data["IsTecnico"] = role == "TECNICO"
		data["IsUsuario"] = role == "USUARIO"
		data["IsSenador"] = role == "SENADOR"
		data["IsFuncionario"] = role == "FUNCIONARIO"

		data["CanManageSystem"] = role == "ADMIN" || role == "TECNICO"
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

func SetSuccessMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "success")
	session.Save()
}

func SetErrorMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "error")
	session.Save()
}
