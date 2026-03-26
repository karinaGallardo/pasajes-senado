package utils

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"time"

	// csrf "github.com/utrack/gin-csrf"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"regexp"
)

var bootTime = time.Now().Unix()

// Render procesa plantillas HTML inyectando tokens CSRF, contexto del usuario, roles y mensajes flash.
func Render(c *gin.Context, templateName string, data gin.H) {
	if data == nil {
		data = gin.H{}
	}

	data["CurrentYear"] = time.Now().Year()
	data["VapidPublicKey"] = viper.GetString("VAPID_PUBLIC_KEY")

	// Detección simple de móvil para ajustes de UI
	ua := c.GetHeader("User-Agent")
	isMobile := false
	mobileRegex := regexp.MustCompile(`(?i)Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini`)
	if mobileRegex.MatchString(ua) {
		isMobile = true
	}
	data["IsMobile"] = isMobile

	// Estrategia de Cache Busting:
	// - En Desarrollo: Cambia en cada render (máxima frescura)
	// - En Producción: Cambia solo al reiniciar el servidor (mejor performance)
	if viper.GetString("ENV") != "production" {
		data["StaticVersion"] = time.Now().Unix()
	} else {
		data["StaticVersion"] = bootTime
	}

	// data["csrf_token"] = csrf.GetToken(c)

	authUser := appcontext.AuthUser(c)
	if authUser != nil {
		data["AuthUser"] = authUser
		role := ""
		if authUser.Rol != nil {
			role = authUser.Rol.Codigo
		}
		data["IsAdmin"] = role == "ADMIN"
		data["IsResponsable"] = role == "RESPONSABLE"
		data["IsUsuario"] = role == "USUARIO"
		data["IsSenador"] = role == "SENADOR"
		data["IsFuncionario"] = role == "FUNCIONARIO"

		data["CanManageSystem"] = role == "ADMIN" || role == "RESPONSABLE"
		data["CanManageUsers"] = role == "ADMIN"
	}

	session := sessions.Default(c)
	success := session.Flashes("success")
	if len(success) > 0 {
		data["SuccessMessage"] = success[0]
	}

	errors := session.Flashes("error")
	if len(errors) > 0 {
		data["ErrorMessage"] = errors[0]
	}
	session.Save()

	c.HTML(http.StatusOK, templateName, data)
}

// SetSuccessMessage registra un mensaje flash de éxito en la sesión actual.
func SetSuccessMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "success")
	session.Save()
}

// SetErrorMessage registra un mensaje flash de error en la sesión actual.
func SetErrorMessage(c *gin.Context, message string) {
	session := sessions.Default(c)
	session.AddFlash(message, "error")
	session.Save()
}

// IsMobileBrowser detecta si el User-Agent corresponde a un dispositivo móvil.
func IsMobileBrowser(c *gin.Context) bool {
	ua := c.GetHeader("User-Agent")
	mobileRegex := regexp.MustCompile(`(?i)Android|webOS|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini`)
	return mobileRegex.MatchString(ua)
}
