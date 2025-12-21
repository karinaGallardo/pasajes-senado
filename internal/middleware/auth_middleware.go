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
