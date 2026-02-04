package routes

import (
	"net/http"
	"sistema-pasajes/internal/controllers"
	"sistema-pasajes/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	authCtrl := controllers.NewAuthController()
	solicitudCtrl := controllers.NewSolicitudController()
	solicitudDerechoCtrl := controllers.NewSolicitudDerechoController()
	pasajeCtrl := controllers.NewPasajeController()
	dashboardCtrl := controllers.NewDashboardController()
	perfilCtrl := controllers.NewPerfilController()
	usuarioCtrl := controllers.NewUsuarioController()
	cupoCtrl := controllers.NewCupoController()

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

		protected.GET("/cupos/derecho/:senador_user_id/:gestion", cupoCtrl.DerechoByYear)
		protected.GET("/cupos/derecho/:senador_user_id/:gestion/:mes", cupoCtrl.DerechoByMonth)

		protected.GET("/solicitudes", solicitudCtrl.Index)
		// protected.GET("/solicitudes/nueva", solicitudCtrl.Create)
		// protected.GET("/solicitudes/:id", solicitudCtrl.Show)
		// protected.POST("/solicitudes", solicitudCtrl.Store) // Generic Store
		// protected.GET("/solicitudes/check-cupo", solicitudCtrl.CheckCupo)
		// protected.POST("/solicitudes/:id/aprobar", solicitudCtrl.Approve)
		// protected.POST("/solicitudes/:id/rechazar", solicitudCtrl.Reject)
		// protected.GET("/solicitudes/:id/editar", solicitudCtrl.Edit)
		// protected.POST("/solicitudes/:id/actualizar", solicitudCtrl.Update)
		// protected.GET("/solicitudes/:id/print", solicitudCtrl.PrintPV01)

		// Solicitudes Derecho
		protected.GET("/solicitudes/derecho/crear/:item_id/:itinerario_code", solicitudDerechoCtrl.Create)
		protected.GET("/solicitudes/derecho/modal-crear/:item_id/:itinerario_code", solicitudDerechoCtrl.GetCreateModal)
		protected.POST("/solicitudes/derecho", solicitudDerechoCtrl.Store)
		protected.GET("/solicitudes/derecho/:id/detalle", solicitudDerechoCtrl.Show)
		protected.GET("/solicitudes/derecho/:id/editar", solicitudDerechoCtrl.Edit)
		protected.GET("/solicitudes/derecho/:id/modal-editar", solicitudDerechoCtrl.GetEditModal)
		protected.POST("/solicitudes/derecho/:id/actualizar", solicitudDerechoCtrl.Update)
		protected.POST("/solicitudes/derecho/:id/aprobar", solicitudDerechoCtrl.Approve)
		protected.POST("/solicitudes/derecho/:id/revertir-aprobacion", solicitudDerechoCtrl.RevertApproval)
		protected.POST("/solicitudes/derecho/:id/rechazar", solicitudDerechoCtrl.Reject)
		protected.POST("/solicitudes/derecho/:id/items/:item_id/aprobar", solicitudDerechoCtrl.ApproveItem)
		protected.POST("/solicitudes/derecho/:id/items/:item_id/rechazar", solicitudDerechoCtrl.RejectItem)
		protected.GET("/solicitudes/derecho/:id/print", solicitudDerechoCtrl.Print)
		protected.DELETE("/solicitudes/derecho/:id", solicitudDerechoCtrl.Destroy)

		protected.POST("/solicitudes/:id/pasajes", pasajeCtrl.Store)
		protected.GET("/solicitudes/:id/pasajes/nuevo", pasajeCtrl.GetCreateModal)
		protected.POST("/pasajes/update-status", pasajeCtrl.UpdateStatus)
		protected.GET("/pasajes/:id/preview", pasajeCtrl.Preview)
		protected.POST("/pasajes/reprogramar", pasajeCtrl.Reprogramar)
		protected.POST("/pasajes/devolver", pasajeCtrl.Devolver)
		protected.POST("/pasajes/update", pasajeCtrl.Update)
		protected.GET("/pasajes/:id/editar", pasajeCtrl.GetEditModal)
		protected.GET("/pasajes/:id/reprogramar", pasajeCtrl.GetReprogramarModal)
		protected.GET("/pasajes/:id/devolver", pasajeCtrl.GetDevolverModal)
		protected.GET("/pasajes/:id/modal-usado", pasajeCtrl.GetUsadoModal)

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
		adminOnly.Use(middleware.RequireRole("ADMIN", "RESPONSABLE"))
		{
			adminOnly.GET("/usuarios", usuarioCtrl.Index)
			adminOnly.GET("/usuarios/table", usuarioCtrl.Table)
			adminOnly.POST("/usuarios/sync", usuarioCtrl.Sync)
			adminOnly.POST("/usuarios/:id/unblock", usuarioCtrl.Unblock)
			adminOnly.GET("/usuarios/sync-modal", usuarioCtrl.GetSyncModal)
		}
		protected.GET("/usuarios/:id/modal-editar", usuarioCtrl.GetEditModal)
		protected.GET("/usuarios/:id/editar", usuarioCtrl.Edit)
		protected.POST("/usuarios/:id/actualizar", usuarioCtrl.Update)

		sysAdmin := protected.Group("/")
		sysAdmin.Use(middleware.RequireRole("ADMIN", "RESPONSABLE"))
		{
			sysAdmin.GET("/admin/cupos", cupoCtrl.Index)
			sysAdmin.POST("/admin/cupos/generar", cupoCtrl.Generar)
			sysAdmin.GET("/admin/cupos/:id/derechos", cupoCtrl.GetCuposByCupo)
			protected.GET("/admin/cupos/derechos/:id/modal-transferir", cupoCtrl.GetTransferModal)
			protected.POST("/admin/cupos/tomar", cupoCtrl.TomarCupo)
			protected.POST("/admin/cupos/asignar", cupoCtrl.AsignarCupo)
			protected.POST("/admin/cupos/transferir", cupoCtrl.Transferir)
			protected.POST("/admin/cupos/derechos/:id/revertir-transferencia", cupoCtrl.RevertirTransferencia)

			sysAdmin.POST("/admin/cupos/reset", cupoCtrl.Reset)

			aerolineaCtrl := controllers.NewAerolineaController()
			sysAdmin.GET("/admin/aerolineas", aerolineaCtrl.Index)
			sysAdmin.GET("/admin/aerolineas/nueva", aerolineaCtrl.New)
			sysAdmin.POST("/admin/aerolineas", aerolineaCtrl.Store)
			sysAdmin.GET("/admin/aerolineas/:id/editar", aerolineaCtrl.Edit)
			sysAdmin.POST("/admin/aerolineas/:id/actualizar", aerolineaCtrl.Update)
			sysAdmin.POST("/admin/aerolineas/:id/toggle", aerolineaCtrl.Toggle)
			sysAdmin.POST("/admin/aerolineas/:id/delete", aerolineaCtrl.Delete)

			agenciaCtrl := controllers.NewAgenciaController()
			sysAdmin.GET("/admin/agencias", agenciaCtrl.Index)
			sysAdmin.GET("/admin/agencias/nueva", agenciaCtrl.New)
			sysAdmin.POST("/admin/agencias", agenciaCtrl.Store)
			sysAdmin.GET("/admin/agencias/:id/editar", agenciaCtrl.Edit)
			sysAdmin.POST("/admin/agencias/:id/actualizar", agenciaCtrl.Update)
			sysAdmin.POST("/admin/agencias/:id/toggle", agenciaCtrl.Toggle)
			sysAdmin.POST("/admin/agencias/:id/delete", agenciaCtrl.Delete)

			rutaCtrl := controllers.NewRutaController()
			sysAdmin.GET("/admin/rutas", rutaCtrl.Index)
			sysAdmin.POST("/admin/rutas", rutaCtrl.Store)
			sysAdmin.GET("/admin/rutas/modal-contrato", rutaCtrl.GetContractModal)
			sysAdmin.POST("/admin/rutas/contrato", rutaCtrl.AddContract)

			confCtrl := controllers.NewConfiguracionController()
			sysAdmin.GET("/admin/configuracion", confCtrl.Index)
			sysAdmin.POST("/admin/configuracion", confCtrl.Update)
			sysAdmin.POST("/admin/configuracion/test-email", confCtrl.TestEmail)

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

	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "errors/404", gin.H{
			"Title": "Página no encontrada",
		})
	})
}
