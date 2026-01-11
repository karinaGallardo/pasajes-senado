package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/routes"
	"sistema-pasajes/internal/utils"

	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	csrf "github.com/utrack/gin-csrf"
	"github.com/webstradev/gin-pagination/v2/pkg/pagination"
)

func main() {
	configs.ConnectDB()

	isDev := viper.GetString("ENV") != "production"
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()
	r.SetFuncMap(utils.TemplateFuncs())

	r.Use(secure.New(secure.Config{
		IsDevelopment:         isDev,
		SSLRedirect:           false,
		STSSeconds:            315360000,
		STSIncludeSubdomains:  true,
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		BrowserXssFilter:      true,
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://cdn.tailwindcss.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:;",
		IENoOpen:              true,
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		SSLProxyHeaders:       map[string]string{"X-Forwarded-Proto": "https"},
	}))

	sessionSecret := viper.GetString("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "secret-de-emergencia-cambiame"
	}
	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 días
		HttpOnly: true,
		Secure:   !isDev,
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("pasajes_session", store))

	r.Use(pagination.New(
		pagination.WithPageText("page"),
		pagination.WithSizeText("limit"),
		pagination.WithDefaultPage(1),
		pagination.WithDefaultPageSize(15),
		pagination.WithMinPageSize(5),
		pagination.WithMaxPageSize(100),
	))

	r.Use(csrf.Middleware(csrf.Options{
		Secret: sessionSecret,
		ErrorFunc: func(c *gin.Context) {
			if c.GetHeader("HX-Request") == "true" {
				c.String(400, "CSRF token mismatch - Recargue la página")
				c.Abort()
				return
			}
			c.String(400, "CSRF token mismatch. Por favor, recargue la página e intente de nuevo.")
			c.Abort()
		},
	}))

	r.Static("/static", "./web/static")
	r.Static("/uploads", "./uploads")

	var files []string
	err := filepath.Walk("web/templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error finding templates: %v", err)
	}
	r.LoadHTMLFiles(files...)

	routes.SetupRoutes(r)

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"system": "Sistema de Pasajes (Senado)",
		})
	})

	port := viper.GetString("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Servidor iniciando en http://localhost:%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Error iniciando servidor: %v", err)
	}
}
