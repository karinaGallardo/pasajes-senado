package middleware

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, err := c.Cookie("user_id")
		if err != nil || username == "" {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		var user models.Usuario
		result := configs.DB.Where("username = ?", username).First(&user)
		if result.Error != nil {
			c.SetCookie("user_id", "", -1, "/", "", false, true)
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		c.Set("User", user)
		c.Next()
	}
}
