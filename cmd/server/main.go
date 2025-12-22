package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/routes"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	configs.ConnectDB()

	migrateFlag := flag.Bool("migrate", false, "Run database migrations and exit")
	flag.Parse()

	if *migrateFlag {
		configs.ConnectDB()
		log.Println("Ejecutando migraciones (GORM AutoMigrate)...")

		err := configs.DB.AutoMigrate(
			&models.Usuario{},
			&models.Rol{},
			&models.Permiso{},
			&models.Solicitud{},
			&models.Pasaje{},
			&models.Descargo{},
			&models.Ciudad{},
			&models.ConceptoViaje{},
			&models.TipoSolicitud{},
			&models.AmbitoViaje{},
			&models.TipoItinerario{},
			&models.Genero{},
		)
		if err != nil {
			log.Fatalf("Error en migración: %v", err)
		}

		roles := []models.Rol{
			{Codigo: "ADMIN", Nombre: "Administrador del Sistema"},
			{Codigo: "TECNICO", Nombre: "Técnico de Sistema"},
			{Codigo: "USUARIO", Nombre: "Usuario Estándar"},
		}
		for _, r := range roles {
			configs.DB.FirstOrCreate(&r, models.Rol{Codigo: r.Codigo})
		}

		fmt.Println("¡Migraciones completadas!")
		os.Exit(0)
	}

	r := gin.Default()

	store := cookie.NewStore([]byte(viper.GetString("SESSION_SECRET")))
	r.Use(sessions.Sessions("pasajes_session", store))

	r.Static("/static", "./web/static")

	r.LoadHTMLGlob("web/templates/**/*")

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
