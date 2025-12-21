package middleware

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

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
		if err := configs.DB.Preload("Rol").First(&user, "id = ?", userID).Error; err != nil {
			session.Clear()
			session.Save()
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		c.Set("UserID", userID)
		c.Set("User", &user)
		c.Next()
	}
}

func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userVal, exists := c.Get("User")
		if !exists {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		user, ok := userVal.(*models.Usuario)
		if !ok || user.Rol == nil {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		for _, role := range allowedRoles {
			if user.Rol.Codigo == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
