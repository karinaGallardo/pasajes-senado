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
	usuarioCtrl := controllers.NewUsuarioController()

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

		protected.POST("/usuarios/:id/update-origin", usuarioCtrl.UpdateOrigin)

		protected.GET("/perfil", perfilCtrl.Show)

		protected.GET("/solicitudes", solicitudCtrl.Index)
		protected.GET("/solicitudes/nueva", solicitudCtrl.Create)
		protected.GET("/solicitudes/derecho/nueva/:id", solicitudCtrl.CreateDerecho)
		protected.POST("/solicitudes", solicitudCtrl.Store)
		protected.GET("/solicitudes/:id", solicitudCtrl.Show)
		protected.GET("/solicitudes/:id/print", solicitudCtrl.PrintPV01)
		protected.GET("/solicitudes/check-cupo", solicitudCtrl.CheckCupo)
		protected.POST("/solicitudes/:id/aprobar", solicitudCtrl.Approve)
		protected.POST("/solicitudes/:id/rechazar", solicitudCtrl.Reject)
		protected.GET("/solicitudes/:id/editar", solicitudCtrl.Edit)
		protected.POST("/solicitudes/:id/actualizar", solicitudCtrl.Update)

		protected.POST("/solicitudes/:id/pasajes", pasajeCtrl.Store)

		viaticoCtrl := controllers.NewViaticoController()
		protected.GET("/solicitudes/:id/viaticos/nuevo", viaticoCtrl.Create)
		protected.POST("/solicitudes/:id/viaticos", viaticoCtrl.Store)
		protected.GET("/viaticos/:id/print", viaticoCtrl.Print)

		descargoCtrl := controllers.NewDescargoController()
		protected.GET("/descargos", descargoCtrl.Index)
		protected.GET("/descargos/nuevo", descargoCtrl.Create)
		protected.POST("/descargos", descargoCtrl.Store)
		protected.GET("/descargos/:id", descargoCtrl.Show)
		protected.POST("/descargos/:id/aprobar", descargoCtrl.Approve)

		compCtrl := controllers.NewCompensacionController()
		protected.GET("/compensaciones", compCtrl.Index)
		protected.GET("/compensaciones/nueva", compCtrl.Create)
		protected.POST("/compensaciones", compCtrl.Store)

		catalogoCtrl := controllers.NewCatalogoController()
		protected.GET("/catalogos/tipos", catalogoCtrl.GetTipos)
		protected.GET("/catalogos/ambitos", catalogoCtrl.GetAmbitos)

		adminOnly := protected.Group("/")
		adminOnly.Use(middleware.RequireRole("ADMIN"))
		{
			adminOnly.GET("/usuarios", usuarioCtrl.Index)
			adminOnly.GET("/usuarios/table", usuarioCtrl.Table)
			adminOnly.POST("/usuarios/sync", usuarioCtrl.Sync)
			adminOnly.GET("/usuarios/:id/editar", usuarioCtrl.Edit)
			adminOnly.POST("/usuarios/:id", usuarioCtrl.Update)
		}

		sysAdmin := protected.Group("/")
		sysAdmin.Use(middleware.RequireRole("ADMIN", "TECNICO"))
		{
			cupoCtrl := controllers.NewCupoController()
			sysAdmin.GET("/admin/cupos", cupoCtrl.Index)
			sysAdmin.POST("/admin/cupos/generar", cupoCtrl.Generar)
			sysAdmin.GET("/admin/cupos/:id/vouchers", cupoCtrl.GetVouchersByCupo)
			sysAdmin.POST("/admin/cupos/transferir", cupoCtrl.Transferir)
			sysAdmin.POST("/admin/cupos/reset", cupoCtrl.Reset)

			provCtrl := controllers.NewProveedorController()
			sysAdmin.GET("/admin/proveedores", provCtrl.Index)
			sysAdmin.POST("/admin/aerolineas", provCtrl.CreateAerolinea)
			sysAdmin.POST("/admin/aerolineas/:id/toggle", provCtrl.ToggleAerolinea)
			sysAdmin.POST("/admin/agencias", provCtrl.CreateAgencia)
			sysAdmin.POST("/admin/agencias/:id/toggle", provCtrl.ToggleAgencia)

			rutaCtrl := controllers.NewRutaController()
			sysAdmin.GET("/admin/rutas", rutaCtrl.Index)
			sysAdmin.POST("/admin/rutas", rutaCtrl.Store)
			sysAdmin.POST("/admin/rutas/contrato", rutaCtrl.AddContract)

			confCtrl := controllers.NewConfiguracionController()
			sysAdmin.GET("/admin/configuracion", confCtrl.Index)
			sysAdmin.POST("/admin/configuracion", confCtrl.Update)

			catCompCtrl := controllers.NewCategoriaCompensacionController()
			sysAdmin.GET("/admin/compensaciones/categorias", catCompCtrl.Index)
			sysAdmin.POST("/admin/compensaciones/categorias", catCompCtrl.Store)
			sysAdmin.POST("/admin/compensaciones/categorias/:id/delete", catCompCtrl.Delete)

			orgCtrl := controllers.NewOrganigramaController()
			sysAdmin.GET("/admin/cargos", orgCtrl.IndexCargos)
			sysAdmin.POST("/admin/cargos", orgCtrl.StoreCargo)
			sysAdmin.POST("/admin/cargos/:id/delete", orgCtrl.DeleteCargo)

			sysAdmin.GET("/admin/oficinas", orgCtrl.IndexOficinas)
			sysAdmin.POST("/admin/oficinas", orgCtrl.StoreOficina)
			sysAdmin.POST("/admin/oficinas/:id/delete", orgCtrl.DeleteOficina)
		}
	}
}
