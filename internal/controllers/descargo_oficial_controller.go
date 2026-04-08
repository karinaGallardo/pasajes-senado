package controllers

import (
	"fmt"
	"log"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type DescargoOficialController struct {
	descargoService        *services.DescargoService
	descargoOficialService *services.DescargoOficialService
	solicitudService       *services.SolicitudService
	reportService          *services.ReportService
	peopleService          *services.PeopleService
	configService          *services.ConfiguracionService
}

func NewDescargoOficialController(
	descargoService *services.DescargoService,
	descargoOficialService *services.DescargoOficialService,
	solicitudService *services.SolicitudService,
	destinoService *services.DestinoService,
	reportService *services.ReportService,
	peopleService *services.PeopleService,
	configService *services.ConfiguracionService,
) *DescargoOficialController {
	return &DescargoOficialController{
		descargoService:        descargoService,
		descargoOficialService: descargoOficialService,
		solicitudService:       solicitudService,
		reportService:          reportService,
		peopleService:          peopleService,
		configService:          configService,
	}
}

func (ctrl *DescargoOficialController) Store(c *gin.Context) {
	solicitudID := c.Param("id")
	if solicitudID == "" {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	// Auto-crear si no existe, o recuperar si ya existe
	descargo, err := ctrl.descargoOficialService.AutoCreateFromSolicitud(c.Request.Context(), solicitud, authUser.ID)
	if err != nil {
		log.Printf("Error en auto-creación de descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	// Redirigir siempre a edición
	c.Redirect(http.StatusFound, "/descargos/oficial/"+descargo.ID+"/editar")
}

func (ctrl *DescargoOficialController) Show(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	tramosIdaOrig := make([]models.DescargoTramo, 0)
	tramosIdaRepro := make([]models.DescargoTramo, 0)
	tramosVueltaOrig := make([]models.DescargoTramo, 0)
	tramosVueltaRepro := make([]models.DescargoTramo, 0)

	for _, t := range descargo.Tramos {
		if strings.HasPrefix(string(t.Tipo), "IDA") {
			if t.IsReprogramacion() {
				tramosIdaRepro = append(tramosIdaRepro, t)
			} else {
				tramosIdaOrig = append(tramosIdaOrig, t)
			}
		} else if strings.HasPrefix(string(t.Tipo), "VUELTA") {
			if t.IsReprogramacion() {
				tramosVueltaRepro = append(tramosVueltaRepro, t)
			} else {
				tramosVueltaOrig = append(tramosVueltaOrig, t)
			}
		}
	}

	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	authUser := appcontext.AuthUser(c)
	descargo.HydratePermissions(authUser)
	if descargo.Solicitud != nil {
		descargo.Solicitud.HydratePermissions(authUser)
	}

	utils.Render(c, "descargo/oficial/show", gin.H{
		"Title":                     "Detalle de Descargo (Oficial)",
		"Descargo":                  descargo,
		"TramosIdaOriginales":       tramosIdaOrig,
		"TramosIdaReprogramados":    tramosIdaRepro,
		"TramosVueltaOriginales":    tramosVueltaOrig,
		"TramosVueltaReprogramados": tramosVueltaRepro,
		"BancoCuenta":               bancoCuenta,
		"BancoNombre":               bancoNombre,
		"User":                      authUser,
	})
}

func (ctrl *DescargoOficialController) Edit(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	authUser := appcontext.AuthUser(c)
	descargo.HydratePermissions(authUser)
	if descargo.Solicitud != nil {
		descargo.Solicitud.HydratePermissions(authUser)
	}

	// 1. Verificación Maestra de Permisos (Estado + Rol/Autoría)
	if !descargo.Permissions.CanEdit {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=SinPermisoEdicion")
		return
	}

	// 2. Sincronización Proactiva (Si hay nuevos pasajes tras la creación)
	if descargo.Solicitud != nil {
		// Pasamos el puntero; el servicio se encarga de persistir y el Slice se actualiza
		if err := ctrl.descargoOficialService.SyncItineraryFromSolicitud(c.Request.Context(), descargo, descargo.Solicitud); err == nil {
			// Recargamos solo si es vital para asegurar que los nuevos DescargoTramo tengan sus Preloads (Ruta, etc.)
			descargo, _ = ctrl.descargoService.GetByID(c.Request.Context(), id)
			descargo.HydratePermissions(authUser)
		}
	}

	tramosIdaOrig := make([]models.DescargoTramo, 0)
	tramosIdaRepro := make([]models.DescargoTramo, 0)
	tramosVueltaOrig := make([]models.DescargoTramo, 0)
	tramosVueltaRepro := make([]models.DescargoTramo, 0)

	for _, t := range descargo.Tramos {
		if strings.HasPrefix(string(t.Tipo), "IDA") {
			if t.IsReprogramacion() {
				tramosIdaRepro = append(tramosIdaRepro, t)
			} else {
				tramosIdaOrig = append(tramosIdaOrig, t)
			}
		} else if strings.HasPrefix(string(t.Tipo), "VUELTA") {
			if t.IsReprogramacion() {
				tramosVueltaRepro = append(tramosVueltaRepro, t)
			} else {
				tramosVueltaOrig = append(tramosVueltaOrig, t)
			}
		}
	}

	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	utils.Render(c, "descargo/oficial/edit", gin.H{
		"Title":                     "Editar Descargo (Oficial)",
		"Descargo":                  descargo,
		"Solicitud":                 descargo.Solicitud,
		"TramosIdaOriginales":       tramosIdaOrig,
		"TramosIdaReprogramados":    tramosIdaRepro,
		"TramosVueltaOriginales":    tramosVueltaOrig,
		"TramosVueltaReprogramados": tramosVueltaRepro,
		"BancoCuenta":               bancoCuenta,
		"BancoNombre":               bancoNombre,
		"User":                      authUser,
	})
}

func (ctrl *DescargoOficialController) Update(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	descargo.HydratePermissions(authUser)
	if !descargo.Permissions.CanEdit {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=EstadoNoPermitido")
		return
	}

	var req dtos.CreateDescargoRequest
	if err := req.Bind(c); err != nil {
		log.Printf("[ERROR] Bind error en Descargo Oficial (ID: %s): %v", id, err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"/editar?error=DatosInvalidos")
		return
	}

	// Delegar recolección de archivos a sus respectivos dueños
	pasesAbordoPaths := utils.ExtractDescargoFiles(c, req.TramoID)
	terrestrePaths := utils.ExtractTerrestreFiles(c, req.TransporteTerrestreID)
	anexoPaths := utils.ExtractDescargoAnexos(c, id)

	if err := ctrl.descargoOficialService.UpdateOficial(c.Request.Context(), id, req, authUser.ID, pasesAbordoPaths, terrestrePaths, anexoPaths); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"/editar?error=ErrorActualizacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"/editar")
}

func (ctrl *DescargoOficialController) Print(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	personaView, _ := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), descargo.Solicitud.Usuario.CI)
	pdf, err := ctrl.reportService.GeneratePV06Complete(c.Request.Context(), descargo, personaView)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generando PDF")
		return
	}

	disposition := "inline"
	if utils.IsMobileBrowser(c) {
		disposition = "attachment"
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"FORM-PV06-%s.pdf\"", disposition, descargo.ID))
	c.Writer.Write(pdf)
}

func (ctrl *DescargoOficialController) Preview(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    "Previsualización Formulario PV-06",
		"FilePath": fmt.Sprintf("/descargos/oficial/%s/imprimir", id),
		"IsPDF":    true,
	})
}

func (ctrl *DescargoOficialController) Submit(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	descargo, _ := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanSubmit {
			c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=SinPermisoEnvio")
			return
		}
	}

	if err := ctrl.descargoService.Submit(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error enviando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorEnvio")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) Approve(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	descargo, _ := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanApprove {
			c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=SinPermisoAprobacion")
			return
		}
	}

	if err := ctrl.descargoService.Approve(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error aprobando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorAprobacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) Reject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	observaciones := c.PostForm("observaciones")

	descargo, _ := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanReject {
			c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=SinPermisoRechazo")
			return
		}
	}

	if err := ctrl.descargoService.Reject(c.Request.Context(), id, authUser.ID, observaciones); err != nil {
		log.Printf("Error rechazando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorRechazo")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) RevertApproval(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}

	descargo, _ := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanRevert {
			c.String(http.StatusForbidden, "No tiene permisos para revertir este descargo")
			return
		}
	}

	if err := ctrl.descargoService.RevertToDraft(c.Request.Context(), id, authUser.ID); err != nil {
		c.String(http.StatusInternalServerError, "Error revirtiendo aprobación: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) NuevaFila(c *gin.Context) {
	tipo := c.Query("tipo")
	solicitudItemID := c.Query("solicitud_item_id")
	index := fmt.Sprintf("new_%d", time.Now().UnixNano())

	c.HTML(http.StatusOK, "descargo/components/tramo_fila_oficial", gin.H{
		"Tipo": tipo,
		"Tramo": models.DescargoTramo{
			BaseModel:       models.BaseModel{ID: index},
			Tipo:            models.TipoDescargoTramo(tipo),
			SolicitudItemID: &solicitudItemID,
			EsDevolucion:    false,
			EsModificacion:  false,
		},
	})
}

func (ctrl *DescargoOficialController) GetModalLiquidar(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	utils.Render(c, "descargo/components/modal_liquidar", gin.H{
		"Descargo":   descargo,
		"ActionURL":  fmt.Sprintf("/descargos/oficial/%s/liquidar", id),
		"csrf_token": c.GetString("csrf_token"),
	})
}

func (ctrl *DescargoOficialController) Liquidar(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var req dtos.LiquidarDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Error bind liquidación oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=MontoInvalido")
		return
	}

	var montos []float64
	for _, mStr := range req.CostosUtilizacion {
		m, _ := strconv.ParseFloat(mStr, 64)
		montos = append(montos, m)
	}

	if err := ctrl.descargoService.Liquidate(c.Request.Context(), id, req.PasajeIDs, montos, authUser.ID); err != nil {
		log.Printf("Error liquidando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorLiquidacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) GetModalPago(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	utils.Render(c, "descargo/components/modal_reportar_pago", gin.H{
		"Descargo":   descargo,
		"ActionURL":  "/descargos/oficial/" + id + "/pago",
		"csrf_token": c.GetString("csrf_token"),
	})
}

func (ctrl *DescargoOficialController) ReportarPago(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	file, err := c.FormFile("comprobante")
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ArchivoRequerido")
		return
	}

	// Guardar el archivo de comprobante
	timestamp := time.Now().UnixNano()
	savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pagos", fmt.Sprintf("pago_oficial_%d_", timestamp))
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorArchivo")
		return
	}

	if err := ctrl.descargoService.ReportPayment(c.Request.Context(), id, savedPath, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorProceso")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) Finalize(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.Finalize(c.Request.Context(), id, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorCierre")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) RevertLiquidation(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertLiquidation(c.Request.Context(), id, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorReversion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) RevertPayment(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertPayment(c.Request.Context(), id, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorReversion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) RevertFinalization(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertFinalization(c.Request.Context(), id, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=ErrorReversion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}
