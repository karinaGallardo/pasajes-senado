package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/routes"

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
			&models.Solicitud{},
			&models.Pasaje{},
			&models.Descargo{},
			&models.Ciudad{},
		)
		if err != nil {
			log.Fatalf("Error en migración: %v", err)
		}

		fmt.Println("¡Migraciones completadas!")
		os.Exit(0)
	}

	r := gin.Default()

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
