package routes

import (
	"sistema-pasajes/internal/controllers"
	"sistema-pasajes/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	authCtrl := controllers.NewAuthController()
	solicitudCtrl := controllers.NewSolicitudController()
	pasajeCtrl := controllers.NewPasajeController()
	dashboardCtrl := controllers.NewDashboardController()
	perfilCtrl := controllers.NewPerfilController()

	r.GET("/auth/login", authCtrl.ShowLogin)
	r.POST("/auth/login", authCtrl.Login)
	r.GET("/auth/logout", authCtrl.Logout)

	protected := r.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		protected.GET("/", func(c *gin.Context) {
			c.Redirect(302, "/dashboard")
		})
		protected.GET("/dashboard", dashboardCtrl.Index)

		protected.GET("/perfil", perfilCtrl.Show)

		protected.GET("/solicitudes", solicitudCtrl.Index)
		protected.GET("/solicitudes/nueva", solicitudCtrl.Create)
		protected.POST("/solicitudes", solicitudCtrl.Store)
		protected.GET("/solicitudes/:id", solicitudCtrl.Show)
		protected.GET("/solicitudes/:id/print", solicitudCtrl.PrintPV01)
		protected.GET("/solicitudes/check-cupo", solicitudCtrl.CheckCupo)
		protected.POST("/solicitudes/:id/aprobar", solicitudCtrl.Approve)
		protected.POST("/solicitudes/:id/rechazar", solicitudCtrl.Reject)
		protected.GET("/solicitudes/:id/editar", solicitudCtrl.Edit)
		protected.POST("/solicitudes/:id/actualizar", solicitudCtrl.Update)

		protected.POST("/solicitudes/:id/pasajes", pasajeCtrl.Store)

		descargoCtrl := controllers.NewDescargoController()
		protected.GET("/descargos", descargoCtrl.Index)
		protected.GET("/descargos/nuevo", descargoCtrl.Create)
		protected.POST("/descargos", descargoCtrl.Store)
		protected.GET("/descargos/:id", descargoCtrl.Show)
		protected.POST("/descargos/:id/aprobar", descargoCtrl.Approve)

		catalogoCtrl := controllers.NewCatalogoController()
		protected.GET("/catalogos/tipos", catalogoCtrl.GetTipos)
		protected.GET("/catalogos/ambitos", catalogoCtrl.GetAmbitos)

		adminOnly := protected.Group("/")
		adminOnly.Use(middleware.RequireRole("ADMIN"))
		{
			usuarioCtrl := controllers.NewUsuarioController()
			adminOnly.GET("/usuarios", usuarioCtrl.Index)
			adminOnly.GET("/usuarios/table", usuarioCtrl.Table)
			adminOnly.GET("/usuarios/:id/editar", usuarioCtrl.Edit)
			adminOnly.POST("/usuarios/:id", usuarioCtrl.Update)
		}
	}
}
