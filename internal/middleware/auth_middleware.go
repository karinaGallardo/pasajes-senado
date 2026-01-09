package middleware

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"slices"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")

		if userID == nil {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		var user models.Usuario
		if err := configs.DB.Preload("Rol").
			Preload("Origen").
			Preload("Origen.Departamento").
			Preload("Departamento").
			Preload("Encargado").
			First(&user, "id = ?", userID).Error; err != nil {
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		appcontext.SetUser(c, &user)

		c.Next()
	}
}

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := appcontext.CurrentUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if user.Rol == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		if slices.Contains(allowedRoles, user.Rol.Codigo) {
			c.Next()
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
