package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/routes"
	"sistema-pasajes/internal/utils"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	configs.ConnectDB()

	r := gin.Default()

	r.SetFuncMap(utils.TemplateFuncs())

	store := cookie.NewStore([]byte(viper.GetString("SESSION_SECRET")))
	r.Use(sessions.Sessions("pasajes_session", store))

	r.Static("/static", "./web/static")

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
