package controllers

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/services"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	authService *services.AuthService
}

func NewAuthController() *AuthController {
	db := configs.DB
	mongoChat := configs.MongoChat
	mongoRRHH := configs.MongoRRHH
	return &AuthController{
		authService: services.NewAuthService(db, mongoChat, mongoRRHH),
	}
}

func (ac *AuthController) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/login.html", gin.H{
		"title": "Iniciar Sesión",
	})
}

func (ac *AuthController) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := ac.authService.AuthenticateAndSync(username, password)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "auth/login.html", gin.H{
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
