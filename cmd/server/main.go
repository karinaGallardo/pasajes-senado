package main

import (
	"fmt"
	"html/template"
	"log"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/routes"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func main() {
	configs.ConnectDB()

	r := gin.Default()

	r.SetFuncMap(template.FuncMap{
		"add": func(a, b float64) float64 {
			return a + b
		},
		"inc": func(i int) int {
			return i + 1
		},
		"dt": func(t *time.Time) string {
			if t == nil {
				return "-"
			}
			return t.Format("02/01/2006")
		},
		"df": func(ini, fin *time.Time) string {
			if ini == nil || fin == nil {
				return "-"
			}
			return fmt.Sprintf("%s - %s", ini.Format("02/01"), fin.Format("02/01"))
		},
	})

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
