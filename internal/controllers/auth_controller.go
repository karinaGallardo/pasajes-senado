package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
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
	var req dtos.LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.Render(c, "auth/login", gin.H{
			"error": "Credenciales inválidas",
		})
		return
	}

	user, err := ac.authService.Authenticate(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		utils.Render(c, "auth/login", gin.H{
			"error": err.Error(),
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)

	if user.Rol == nil || user.Rol.Codigo == "" {
		utils.Render(c, "auth/login", gin.H{
			"error": "Error: El usuario no tiene un rol asignado en el sistema.",
		})
		return
	}
	roleCode := user.Rol.Codigo
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
