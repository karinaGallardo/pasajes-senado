package routes

import (
	"net/http"
	"sistema-pasajes/internal/app"
	"sistema-pasajes/internal/middleware"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, container *app.Container) {
	authCtrl := container.AuthController
	solicitudCtrl := container.SolicitudController
	solicitudDerechoCtrl := container.SolicitudDerechoController
	solicitudOficialCtrl := container.SolicitudOficialController
	pasajeCtrl := container.PasajeController
	dashboardCtrl := container.DashboardController
	perfilCtrl := container.PerfilController
	usuarioCtrl := container.UsuarioController
	senadorCtrl := container.SenadorController
	funcionarioCtrl := container.FuncionarioController
	cupoCtrl := container.CupoController
	descargoDerechoCtrl := container.DescargoDerechoController
	descargoOficialCtrl := container.DescargoOficialController
	compensacionCtrl := container.CompensacionController
	catalogoCtrl := container.CatalogoController
	viaticoCtrl := container.ViaticoController
	aerolineaCtrl := container.AerolineaController
	agenciaCtrl := container.AgenciaController
	rutaCtrl := container.RutaController
	confCtrl := container.ConfiguracionController
	catCompCtrl := container.CatCompensacionController
	orgCtrl := container.OrganigramaController
	catViaticoCtrl := container.CategoriaViaticoController
	notifCtrl := container.NotificationController

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
		protected.GET("/solicitudes/derecho", solicitudCtrl.IndexDerecho)
		protected.GET("/solicitudes/derecho/table", solicitudCtrl.TableDerecho)
		protected.GET("/solicitudes/oficial", solicitudCtrl.IndexOficial)
		protected.GET("/solicitudes/oficial/table", solicitudCtrl.TableOficial)
		protected.GET("/solicitudes/pendientes-descargo", solicitudCtrl.IndexPendientesDescargo)
		protected.GET("/solicitudes/pendientes-descargo/table", solicitudCtrl.TablePendientesDescargo)
		protected.GET("/api/solicitudes/pending-stats", solicitudCtrl.GetPendingStats)

		// Solicitudes Derecho
		protected.GET("/solicitudes/derecho/modal-crear/:item_id/:itinerario_code", solicitudDerechoCtrl.GetCreateModal)
		protected.GET("/solicitudes/derecho/:id/detalle", solicitudDerechoCtrl.Show)
		protected.GET("/solicitudes/derecho/:id/modal-editar", solicitudDerechoCtrl.GetEditModal)
		protected.GET("/solicitudes/derecho/:id/print", solicitudDerechoCtrl.Print)

		// Descargos Derecho
		protected.GET("/descargos/derecho/nuevo/:id", descargoDerechoCtrl.Create)
		protected.POST("/descargos/derecho", descargoDerechoCtrl.Store)
		protected.GET("/descargos/derecho/:id", descargoDerechoCtrl.Show)
		protected.GET("/descargos/derecho/:id/editar", descargoDerechoCtrl.Edit)
		protected.POST("/descargos/derecho/:id/actualizar", descargoDerechoCtrl.Update)
		protected.GET("/descargos/derecho/:id/imprimir", descargoDerechoCtrl.Print)
		protected.GET("/descargos/derecho/:id/previsualizar", descargoDerechoCtrl.Preview)
		protected.POST("/descargos/derecho/:id/aprobar", descargoDerechoCtrl.Approve)
		protected.POST("/descargos/derecho/:id/rechazar", descargoDerechoCtrl.Reject)
		protected.POST("/descargos/derecho/:id/enviar", descargoDerechoCtrl.Submit)
		protected.POST("/descargos/derecho/:id/revertir-aprobacion", descargoDerechoCtrl.RevertApproval)

		protected.POST("/solicitudes/derecho/:id/actualizar", solicitudDerechoCtrl.Update)
		protected.POST("/solicitudes/derecho/:id/aprobar", solicitudDerechoCtrl.Approve)
		protected.POST("/solicitudes/derecho/:id/revertir-aprobacion", solicitudDerechoCtrl.RevertApproval)
		protected.POST("/solicitudes/derecho/:id/rechazar", solicitudDerechoCtrl.Reject)
		protected.POST("/solicitudes/derecho/:id/items/:item_id/aprobar", solicitudDerechoCtrl.ApproveItem)
		protected.POST("/solicitudes/derecho/:id/items/:item_id/revertir-aprobacion", solicitudDerechoCtrl.RevertApprovalItem)
		protected.POST("/solicitudes/derecho/:id/items/:item_id/rechazar", solicitudDerechoCtrl.RejectItem)
		protected.POST("/solicitudes/derecho", solicitudDerechoCtrl.Store)
		protected.DELETE("/solicitudes/derecho/:id", solicitudDerechoCtrl.Destroy)

		// Solicitudes Oficial
		protected.GET("/solicitudes/oficial/modal-crear", solicitudOficialCtrl.GetCreateModal)
		protected.GET("/solicitudes/oficial/:id/detalle", solicitudOficialCtrl.Show)
		protected.GET("/solicitudes/oficial/:id/modal-editar", solicitudOficialCtrl.GetEditModal)
		protected.GET("/solicitudes/oficial/:id/print", solicitudOficialCtrl.Print)

		// Descargos Oficial
		protected.GET("/descargos/oficial/nuevo/:id", descargoOficialCtrl.Create)
		protected.POST("/descargos/oficial", descargoOficialCtrl.Store)
		protected.GET("/descargos/oficial/:id", descargoOficialCtrl.Show)
		protected.GET("/descargos/oficial/:id/editar", descargoOficialCtrl.Edit)
		protected.POST("/descargos/oficial/:id/actualizar", descargoOficialCtrl.Update)
		protected.GET("/descargos/oficial/:id/imprimir", descargoOficialCtrl.Print)
		protected.GET("/descargos/oficial/:id/previsualizar", descargoOficialCtrl.Preview)
		protected.POST("/descargos/oficial/:id/aprobar", descargoOficialCtrl.Approve)
		protected.POST("/descargos/oficial/:id/rechazar", descargoOficialCtrl.Reject)
		protected.POST("/descargos/oficial/:id/enviar", descargoOficialCtrl.Submit)
		protected.POST("/descargos/oficial/:id/revertir-aprobacion", descargoOficialCtrl.RevertApproval)

		protected.POST("/solicitudes/oficial/:id/actualizar", solicitudOficialCtrl.Update)
		protected.POST("/solicitudes/oficial", solicitudOficialCtrl.Store)
		protected.POST("/solicitudes/oficial/:id/aprobar", solicitudOficialCtrl.Approve)
		protected.POST("/solicitudes/oficial/:id/revertir-aprobacion", solicitudOficialCtrl.RevertApproval)
		protected.POST("/solicitudes/oficial/:id/rechazar", solicitudOficialCtrl.Reject)
		protected.POST("/solicitudes/oficial/:id/items/:item_id/aprobar", solicitudOficialCtrl.ApproveItem)
		protected.POST("/solicitudes/oficial/:id/items/:item_id/revertir-aprobacion", solicitudOficialCtrl.RevertApprovalItem)
		protected.POST("/solicitudes/oficial/:id/items/:item_id/rechazar", solicitudOficialCtrl.RejectItem)

		protected.POST("/solicitudes/:id/pasajes", pasajeCtrl.Store)
		protected.GET("/solicitudes/:id/pasajes/nuevo", pasajeCtrl.GetCreateModal)
		protected.POST("/pasajes/update-status", pasajeCtrl.UpdateStatus)
		protected.GET("/pasajes/:id/preview", pasajeCtrl.Preview)
		protected.POST("/solicitud-items/reprogramar", solicitudDerechoCtrl.ReprogramarItem)
		protected.POST("/pasajes/devolver", pasajeCtrl.Devolver)
		protected.POST("/pasajes/update", pasajeCtrl.Update)
		protected.GET("/pasajes/:id/editar", pasajeCtrl.GetEditModal)
		protected.GET("/solicitud-items/:id/reprogramar", solicitudDerechoCtrl.GetReprogramarModalSolicitudItem)
		protected.GET("/pasajes/:id/devolver", pasajeCtrl.GetDevolverModal)
		protected.GET("/pasajes/:id/modal-usado", pasajeCtrl.GetUsadoModal)

		protected.GET("/viaticos", viaticoCtrl.Index)
		protected.GET("/solicitudes/:id/viaticos/nuevo", viaticoCtrl.Create)
		protected.POST("/solicitudes/:id/viaticos", viaticoCtrl.Store)
		protected.GET("/viaticos/:id/print", viaticoCtrl.Print)

		// Descargos Comunes (Manejados por Derecho Controller para conveniencia)
		protected.GET("/descargos", descargoDerechoCtrl.Index)
		protected.GET("/descargos/table", descargoDerechoCtrl.Table)
		protected.GET("/preview-file", descargoDerechoCtrl.PreviewFile)

		protected.GET("/compensaciones", compensacionCtrl.Index)
		protected.GET("/compensaciones/nueva", compensacionCtrl.Create)
		protected.POST("/compensaciones", compensacionCtrl.Store)
		protected.GET("/catalogos/tipos", catalogoCtrl.GetTipos)
		protected.GET("/catalogos/ambitos", catalogoCtrl.GetAmbitos)

		adminOnly := protected.Group("/")
		adminOnly.Use(middleware.RequireRole("ADMIN", "RESPONSABLE"))
		{
			adminOnly.GET("/usuarios/senadores", senadorCtrl.Index)
			adminOnly.GET("/usuarios/senadores/table", senadorCtrl.Table)
			adminOnly.POST("/usuarios/senadores/sync", senadorCtrl.Sync)
			adminOnly.GET("/usuarios/senadores/sync-modal", senadorCtrl.GetSyncModal)

			adminOnly.GET("/usuarios/funcionarios", funcionarioCtrl.Index)
			adminOnly.GET("/usuarios/funcionarios/table", funcionarioCtrl.Table)
			adminOnly.POST("/usuarios/funcionarios/sync", funcionarioCtrl.Sync)
			adminOnly.GET("/usuarios/funcionarios/sync-modal", funcionarioCtrl.GetSyncModal)

			adminOnly.POST("/usuarios/:id/unblock", usuarioCtrl.Unblock)
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

			sysAdmin.GET("/admin/aerolineas", aerolineaCtrl.Index)
			sysAdmin.GET("/admin/aerolineas/nueva", aerolineaCtrl.New)
			sysAdmin.POST("/admin/aerolineas", aerolineaCtrl.Store)
			sysAdmin.GET("/admin/aerolineas/:id/editar", aerolineaCtrl.Edit)
			sysAdmin.POST("/admin/aerolineas/:id/actualizar", aerolineaCtrl.Update)
			sysAdmin.POST("/admin/aerolineas/:id/toggle", aerolineaCtrl.Toggle)
			sysAdmin.POST("/admin/aerolineas/:id/delete", aerolineaCtrl.Delete)

			sysAdmin.GET("/admin/agencias", agenciaCtrl.Index)
			sysAdmin.GET("/admin/agencias/nueva", agenciaCtrl.New)
			sysAdmin.POST("/admin/agencias", agenciaCtrl.Store)
			sysAdmin.GET("/admin/agencias/:id/editar", agenciaCtrl.Edit)
			sysAdmin.POST("/admin/agencias/:id/actualizar", agenciaCtrl.Update)
			sysAdmin.POST("/admin/agencias/:id/toggle", agenciaCtrl.Toggle)
			sysAdmin.POST("/admin/agencias/:id/delete", agenciaCtrl.Delete)

			sysAdmin.GET("/admin/rutas", rutaCtrl.Index)
			sysAdmin.POST("/admin/rutas", rutaCtrl.Store)
			sysAdmin.GET("/admin/rutas/modal-contrato", rutaCtrl.GetContractModal)
			sysAdmin.POST("/admin/rutas/contrato", rutaCtrl.AddContract)
			sysAdmin.POST("/admin/rutas/contrato/:id/delete", rutaCtrl.DeleteContract)

			sysAdmin.GET("/admin/configuracion", confCtrl.Index)
			sysAdmin.POST("/admin/configuracion", confCtrl.Update)
			sysAdmin.POST("/admin/configuracion/test-email", confCtrl.TestEmail)

			sysAdmin.GET("/admin/compensaciones/categorias", catCompCtrl.Index)
			sysAdmin.POST("/admin/compensaciones/categorias", catCompCtrl.Store)
			sysAdmin.POST("/admin/compensaciones/categorias/:id/delete", catCompCtrl.Delete)

			sysAdmin.GET("/admin/cargos", orgCtrl.IndexCargos)
			sysAdmin.POST("/admin/cargos", orgCtrl.StoreCargo)
			sysAdmin.POST("/admin/cargos/:id/delete", orgCtrl.DeleteCargo)

			sysAdmin.GET("/admin/oficinas", orgCtrl.IndexOficinas)
			sysAdmin.POST("/admin/oficinas", orgCtrl.StoreOficina)
			sysAdmin.POST("/admin/oficinas/:id/delete", orgCtrl.DeleteOficina)

			sysAdmin.GET("/admin/viaticos/categorias", catViaticoCtrl.Index)
			sysAdmin.POST("/admin/viaticos/categorias", catViaticoCtrl.Store)
			protected.POST("/admin/viaticos/zonas", catViaticoCtrl.StoreZona)
		}

		protected.GET("/api/notifications/recent", notifCtrl.GetRecent)
		protected.POST("/api/notifications/:id/read", notifCtrl.MarkAsRead)
		protected.POST("/api/notifications/read-all", notifCtrl.MarkAllAsRead)

		protected.GET("/ws/notifications", func(c *gin.Context) {
			services.Hub.HandleWebSocket(c.Writer, c.Request)
		})
	}

	r.NoRoute(func(c *gin.Context) {
		c.HTML(http.StatusNotFound, "errors/404", gin.H{
			"Title": "Página no encontrada",
		})
	})
}
