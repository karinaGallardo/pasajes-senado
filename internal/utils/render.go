package utils

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"

	"github.com/gin-gonic/gin"
)

func Render(c *gin.Context, templateName string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	user := appcontext.CurrentUser(c)
	if user != nil {
		data["User"] = user
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

	c.HTML(http.StatusOK, templateName, data)
}
