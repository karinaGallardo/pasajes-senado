package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/routes"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"sistema-pasajes/internal/worker"

	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/webstradev/gin-pagination/v2/pkg/pagination"
)

func main() {
	configs.ConnectDB()
	services.InitHub()

	// --- Background Workers ---
	workerPool := worker.GetPool()
	workerPool.Start(context.Background())
	defer workerPool.Stop()

	// --- Professional Scheduler (robfig/cron) ---
	c := cron.New(cron.WithLocation(time.Local))
	alertaService := services.NewAlertaService()

	// "0 7 * * 1-5" means: At 07:00 AM, Mon through Fri
	_, err := c.AddFunc("0 7 * * 1-5", func() {
		log.Println("[Scheduler] Ejecutando alertas de descargo programadas (7:00 AM Mon-Fri)...")
		workerPool.Submit(&services.AlertaDescargoJob{Service: alertaService})
	})
	if err != nil {
		log.Printf("[Scheduler] ERROR al programar alertas: %v", err)
	}

	c.Start()
	defer c.Stop()

	log.Println("[Scheduler] Programador iniciado: Alertas diarias Mon-Fri 07:00 AM.")

	itinerarioService := services.NewTipoItinerarioService()
	if err := itinerarioService.EnsureDefaults(context.Background()); err != nil {
		log.Printf("Error seeding itineraries: %v", err)
	}

	isDev := viper.GetString("ENV") != "production"
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.ForwardedByClientIP = true
	r.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "192.168.20.0/16"})

	r.SetFuncMap(utils.TemplateFuncs())

	r.Use(secure.New(secure.Config{
		IsDevelopment:           isDev,
		SSLRedirect:             false,
		STSSeconds:              315360000,
		STSIncludeSubdomains:    true,
		FrameDeny:               false,
		CustomFrameOptionsValue: "SAMEORIGIN",
		ContentTypeNosniff:      true,
		BrowserXssFilter:        true,
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net https://cdn.tailwindcss.com https://cdnjs.cloudflare.com https://unpkg.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://cdnjs.cloudflare.com https://unpkg.com; font-src 'self' https://fonts.gstatic.com https://cdnjs.cloudflare.com; img-src 'self' data: https://cdn.tailwindcss.com; connect-src 'self'; frame-src 'self'; object-src 'self';",
		IENoOpen:                true,
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		SSLProxyHeaders:         map[string]string{"X-Forwarded-Proto": "https"},
	}))

	sessionSecret := viper.GetString("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "secret-de-emergencia-cambiame"
	}

	sessionSecure := viper.GetBool("SESSION_SECURE")
	if !viper.IsSet("SESSION_SECURE") {
		sessionSecure = false
	}

	store := cookie.NewStore([]byte(sessionSecret))
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 días
		HttpOnly: true,
		Secure:   sessionSecure,
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

	// r.Use(csrf.Middleware(csrf.Options{
	// 	Secret: sessionSecret,
	// 	ErrorFunc: func(c *gin.Context) {
	// 		log.Printf("[CSRF ERROR] Method: %s, Path: %s, RemoteAddr: %s, HX-Request: %v",
	// 			c.Request.Method, c.Request.URL.Path, c.ClientIP(), c.GetHeader("HX-Request"))

	// 		if c.GetHeader("HX-Request") == "true" {
	// 			c.String(400, "CSRF token mismatch - Recargue la página")
	// 			c.Abort()
	// 			return
	// 		}
	// 		c.String(400, "CSRF token mismatch. Por favor, recargue la página e intente de nuevo.")
	// 		c.Abort()
	// 	},
	// }))

	r.Static("/static", "./web/static")
	r.Static("/uploads", "./uploads")

	var files []string
	err = filepath.Walk("web/templates", func(path string, info os.FileInfo, err error) error {
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
