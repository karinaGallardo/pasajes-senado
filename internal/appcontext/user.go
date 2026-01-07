package appcontext

import (
	"sistema-pasajes/internal/models"

	"github.com/gin-gonic/gin"
)

func SetUser(c *gin.Context, user *models.Usuario) {
	c.Set("auth_user", user)
}

func CurrentUser(c *gin.Context) *models.Usuario {
	if val, exists := c.Get("auth_user"); exists {
		if u, ok := val.(*models.Usuario); ok {
			return u
		}
	}
	return nil
}
