package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sistema-pasajes/internal/app"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/middleware"
	"sistema-pasajes/internal/routes"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"sistema-pasajes/internal/worker"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/secure"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"github.com/webstradev/gin-pagination/v2/pkg/pagination"
)

func main() {
	// --- POINT 3: Structured Logging (slog) ---
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if viper.GetString("ENV") == "production" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)

	configs.ConnectDB()
	services.InitHub()

	// --- Background Workers ---
	workerPool := worker.GetPool()
	workerPool.Start(context.Background())

	container := app.NewContainer(configs.DB, configs.MongoRRHH, configs.MongoChat)

	// --- Professional Scheduler (robfig/cron) ---
	loc, err := time.LoadLocation("America/La_Paz")
	if err != nil {
		slog.Warn("[Scheduler] Falló al cargar America/La_Paz, usando Local", "error", err)
		loc = time.Local
	}
	c := cron.New(cron.WithLocation(loc))
	alertaService := container.AlertaService

	// "0 9 * * 1-5" means: At 09:00 AM, Mon through Fri
	_, err = c.AddFunc("0 9 * * 1-5", func() {
		slog.Info("[Scheduler] Ejecutando alertas de descargo programadas (9:00 AM Mon-Fri)...")
		workerPool.Submit(&services.AlertaDescargoJob{Service: alertaService})
	})
	if err != nil {
		slog.Error("[Scheduler] Error al programar alertas", "error", err)
	}

	c.Start()
	slog.Info("[Scheduler] Programador iniciado: Alertas diarias Mon-Fri 09:00 AM America/La_Paz.")

	itinerarioService := container.TipoItinerarioService
	if err := itinerarioService.EnsureDefaults(context.Background()); err != nil {
		slog.Error("Error seeding itineraries", "error", err)
	}

	isDev := viper.GetString("ENV") != "production"
	if !isDev {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	r.Use(middleware.MetadataMiddleware())

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

	r.Static("/static", "./web/static")
	r.Static("/uploads", "./uploads")

	// PWA Routes
	r.StaticFile("/sw.js", "./web/static/sw.js")
	r.StaticFile("/manifest.json", "./web/static/img/site.webmanifest")
	r.StaticFile("/favicon.ico", "./web/static/img/favicon.ico")

	var templates []string
	err = filepath.Walk("web/templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			templates = append(templates, path)
		}
		return nil
	})
	if err != nil {
		slog.Error("Error finding templates", "error", err)
		os.Exit(1)
	}
	r.LoadHTMLFiles(templates...)

	// Hardening: Rate Limiting para Login (5 peticiones por minuto por IP)
	loginLimiter := middleware.NewIPRateLimiter(5.0/60.0, 5)

	routes.SetupRoutes(r, container, loginLimiter)

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

	// --- POINT 1: Graceful Shutdown Implementation ---
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		slog.Info("Servidor iniciando", "url", "http://localhost:"+port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Error iniciando servidor", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no parameter) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so no need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Apagando servidor de forma segura...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Servidor forzado a apagarse", "error", err)
	}

	// Stop other services
	c.Stop()
	workerPool.Stop()
	slog.Info("Servidor detenido limpiamente.")
}
