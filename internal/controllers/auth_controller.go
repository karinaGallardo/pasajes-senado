package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	return &AuthController{
		authService: services.NewAuthService(),
	}
}

func (ac *AuthController) ShowLogin(c *gin.Context) {
	utils.Render(c, "auth/login", gin.H{
		"title": "Iniciar Sesión",
	})
}

func (ac *AuthController) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := ac.authService.AuthenticateAndSync(c.Request.Context(), username, password)
	if err != nil {
		utils.Render(c, "auth/login", gin.H{
			"error": err.Error(),
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)

	roleCode := "FUNCIONARIO"
	if user.Rol != nil {
		roleCode = user.Rol.Codigo
	}
	session.Set("role", roleCode)
	session.Set("nombre", user.GetNombreCompleto())
	session.Save()

	c.Redirect(http.StatusFound, "/dashboard")
}

func (ac *AuthController) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/auth/login")
}
