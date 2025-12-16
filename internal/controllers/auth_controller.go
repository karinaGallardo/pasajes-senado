package controllers

import (
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthController struct {
	mongoRepo *repositories.MongoUserRepository
	sqlRepo   *repositories.UsuarioRepository
}

func NewAuthController() *AuthController {
	return &AuthController{
		mongoRepo: repositories.NewMongoUserRepository(),
		sqlRepo:   repositories.NewUsuarioRepository(),
	}
}

func (ctrl *AuthController) ShowLogin(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/login.html", gin.H{
		"Title": "Iniciar Sesión",
	})
}

func (ctrl *AuthController) Login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := strings.TrimSpace(c.PostForm("password"))

	mongoUser, err := ctrl.mongoRepo.FindByUsername(username)
	if err != nil {
		c.HTML(http.StatusUnauthorized, "auth/login.html", gin.H{
			"Error": "Usuario no encontrado (Mongo)",
		})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(mongoUser.Password), []byte(password))
	if err != nil {
		c.HTML(http.StatusUnauthorized, "auth/login.html", gin.H{
			"Error": "Contraseña incorrecta",
		})
		return
	}

	sqlUser := models.Usuario{
		ID:       mongoUser.ID.Hex(),
		Username: mongoUser.Username,
		CI:       mongoUser.CI,
	}

	if err := configs.DB.Where("username = ?", sqlUser.Username).FirstOrCreate(&sqlUser).Error; err != nil {
		configs.DB.Model(&models.Usuario{}).Where("username = ?", sqlUser.Username).Updates(models.Usuario{
			CI: mongoUser.CI,
		})
		configs.DB.Where("username = ?", sqlUser.Username).First(&sqlUser)
	}

	c.SetCookie("user_id", sqlUser.Username, 3600*8, "/", "", false, true)

	c.Redirect(http.StatusFound, "/dashboard")
}

func (ctrl *AuthController) Logout(c *gin.Context) {
	c.SetCookie("user_id", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}
