package controllers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
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

type DescargoDerechoController struct {
	descargoService        *services.DescargoService
	descargoDerechoService *services.DescargoDerechoService
	solicitudService       *services.SolicitudService
	destinoService         *services.DestinoService
	reportService          *services.ReportService
	peopleService          *services.PeopleService
	aerolineaService       *services.AerolineaService
	configService          *services.ConfiguracionService
}

func NewDescargoDerechoController(
	descargoService *services.DescargoService,
	descargoDerechoService *services.DescargoDerechoService,
	solicitudService *services.SolicitudService,
	destinoService *services.DestinoService,
	reportService *services.ReportService,
	peopleService *services.PeopleService,
	aerolineaService *services.AerolineaService,
	usuarioService *services.UsuarioService,
	configService *services.ConfiguracionService,
) *DescargoDerechoController {
	return &DescargoDerechoController{
		descargoService:        descargoService,
		descargoDerechoService: descargoDerechoService,
		solicitudService:       solicitudService,
		destinoService:         destinoService,
		reportService:          reportService,
		peopleService:          peopleService,
		aerolineaService:       aerolineaService,
		configService:          configService,
	}
}

func (ctrl *DescargoDerechoController) Store(c *gin.Context) {
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
	descargo, err := ctrl.descargoDerechoService.AutoCreateFromSolicitud(c.Request.Context(), solicitud, authUser.ID)
	if err != nil {
		log.Printf("Error en auto-creación de descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	// Redirigir siempre a edición
	c.Redirect(http.StatusFound, "/descargos/derecho/"+descargo.ID+"/editar")
}

func (ctrl *DescargoDerechoController) Show(c *gin.Context) {
	id := c.Param("id")
	data, err := ctrl.descargoDerechoService.GetShowData(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	authUser := appcontext.AuthUser(c)
	data.Descargo.HydratePermissions(authUser)
	if data.Descargo.Solicitud != nil {
		data.Descargo.Solicitud.HydratePermissions(authUser)
	}

	configs, _ := ctrl.configService.GetAll(c.Request.Context())
	configMap := make(map[string]string)
	for _, c := range configs {
		configMap[c.Clave] = c.Valor
	}

	utils.Render(c, "descargo/derecho/show", gin.H{
		"Title":      "Detalle de Descargo (Derecho)",
		"Descargo":   data.Descargo,
		"Solicitud":  data.Descargo.Solicitud,
		"Ida":        data.Ida,
		"Vuelta":     data.Vuelta,
		"Config":     configMap,
		"csrf_token": c.GetString("csrf_token"),
	})
}

func (ctrl *DescargoDerechoController) Edit(c *gin.Context) {
	id := c.Param("id")
	data, err := ctrl.descargoDerechoService.GetEditData(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	authUser := appcontext.AuthUser(c)
	data.Descargo.HydratePermissions(authUser)
	if data.Descargo.Solicitud != nil {
		data.Descargo.Solicitud.HydratePermissions(authUser)
	}

	// Verificación Maestra de Permisos
	if !data.Descargo.Permissions.CanEdit {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=SinPermisoEdicion")
		return
	}

	utils.Render(c, "descargo/derecho/edit", gin.H{
		"Title":     "Editar Descargo (Derecho)",
		"Descargo":  data.Descargo,
		"Solicitud": data.Solicitud,
		"Ida":       data.Ida,
		"Vuelta":    data.Vuelta,
	})
}

func (ctrl *DescargoDerechoController) Update(c *gin.Context) {
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
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=EstadoNoPermitido")
		return
	}

	var req dtos.CreateDescargoRequest
	if err := req.Bind(c); err != nil {
		log.Printf("[ERROR] Bind error en Descargo Derecho (ID: %s): %v", id, err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar?error=DatosInvalidos")
		return
	}

	// Delegar recolección de archivos a sus respectivos dueños
	pasesAbordoPaths := utils.ExtractDescargoFiles(c, req.TramoID)

	if err := ctrl.descargoDerechoService.UpdateDerecho(c.Request.Context(), id, req, authUser.ID, pasesAbordoPaths); err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar?error=ErrorActualizacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar")
}

func (ctrl *DescargoDerechoController) Print(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	personaView, _ := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), descargo.Solicitud.Usuario.CI)
	pdfReader, err := ctrl.reportService.GeneratePV05Complete(c.Request.Context(), descargo, personaView)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generando PDF")
		return
	}

	disposition := "inline"
	if utils.IsMobileBrowser(c) {
		disposition = "attachment"
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"FORM-PV05-%s.pdf\"", disposition, descargo.ID))
	c.Writer.Write(pdfReader)
}

func (ctrl *DescargoDerechoController) Preview(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    "Previsualización Formulario PV-05",
		"FilePath": fmt.Sprintf("/descargos/derecho/%s/imprimir", id),
		"IsPDF":    true,
	})
}

func (ctrl *DescargoDerechoController) Index(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.descargoService.GetPaginatedScoped(c.Request.Context(), authUser, page, limit, searchTerm)

	for i := range result.Descargos {
		result.Descargos[i].HydratePermissions(authUser)
	}

	utils.Render(c, "descargo/index", gin.H{
		"Title":    "Bandeja de Descargos",
		"Result":   result,
		"LinkBase": "/descargos",
	})
}

func (ctrl *DescargoDerechoController) Table(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	searchTerm := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, _ := ctrl.descargoService.GetPaginatedScoped(c.Request.Context(), authUser, page, limit, searchTerm)

	for i := range result.Descargos {
		result.Descargos[i].HydratePermissions(authUser)
	}

	utils.Render(c, "descargo/table_descargos", gin.H{
		"Result":   result,
		"LinkBase": "/descargos",
	})
}

func (ctrl *DescargoDerechoController) Submit(c *gin.Context) {
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
			c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=SinPermisoEnvio")
			return
		}
	}

	if err := ctrl.descargoService.Submit(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error enviando descargo derecho: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorEnvio")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) Approve(c *gin.Context) {
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
			c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=SinPermisoAprobacion")
			return
		}
	}

	if err := ctrl.descargoService.Approve(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error aprobando descargo derecho: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorAprobacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) Reject(c *gin.Context) {
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
			c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=SinPermisoRechazo")
			return
		}
	}

	if err := ctrl.descargoService.Reject(c.Request.Context(), id, authUser.ID, observaciones); err != nil {
		log.Printf("Error rechazando descargo derecho: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorRechazo")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RevertApproval(c *gin.Context) {
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

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RawFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.String(http.StatusBadRequest, "Ruta de archivo requerida")
		return
	}

	// Seguridad básica: Impedir que salgan de la carpeta uploads
	if !strings.HasPrefix(path, "uploads/") && !strings.HasPrefix(path, "/uploads/") {
		c.String(http.StatusForbidden, "Acceso denegado a esta ruta")
		return
	}

	cleanPath := strings.TrimPrefix(path, "/")
	if _, err := os.Stat(cleanPath); err != nil {
		c.String(http.StatusNotFound, "Archivo no encontrado físicamente")
		return
	}

	// Servir archivo plano (mantiene mime-type automático)
	c.File(cleanPath)
}

func (ctrl *DescargoDerechoController) PreviewFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.String(http.StatusBadRequest, "Ruta de archivo requerida")
		return
	}

	// Preparamos la URL del recurso puro para el src de la imagen/iframe
	rawFileUrl := "/raw-file?path=" + url.QueryEscape(path)

	lowerPath := strings.ToLower(path)
	isPDF := strings.HasSuffix(lowerPath, ".pdf")
	isImage := strings.HasSuffix(lowerPath, ".jpg") ||
		strings.HasSuffix(lowerPath, ".jpeg") ||
		strings.HasSuffix(lowerPath, ".png") ||
		strings.HasSuffix(lowerPath, ".gif") ||
		strings.HasSuffix(lowerPath, ".webp")

	title := "Previsualización de Documento"
	if isImage {
		title = "Previsualización de Imagen"
	}

	utils.Render(c, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":                 title,
		"FilePath":              rawFileUrl, // Ahora apunta al endpoint puro
		"IsPDF":                 isPDF,
		"IsImage":               isImage,
		"InfoRuta":              c.Query("ruta"),
		"InfoFecha":             c.Query("fecha"),
		"InfoBillete":           c.Query("billete"),
		"InfoVuelo":             c.Query("vuelo"),
		"InfoTramoRegistrado":   c.Query("info_tramo_registrado"),
		"InfoFechaRegistrada":   c.Query("info_fecha_registrada"),
		"InfoBilleteRegistrado": c.Query("info_billete_registrado"),
		"InfoPaseRegistrado":    c.Query("info_pase_registrado"),
		"IsMobile":              utils.IsMobileBrowser(c),
	})
}

func (ctrl *DescargoDerechoController) NuevaFila(c *gin.Context) {
	tipo := c.Query("tipo")
	solicitudItemID := c.Query("solicitud_item_id")
	index := fmt.Sprintf("new_%d", time.Now().UnixNano())

	c.HTML(http.StatusOK, "descargo/components/tramo_fila_derecho", gin.H{
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

func (ctrl *DescargoDerechoController) UploadSingle(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No se recibió ningún archivo"})
		return
	}

	// Guardar el archivo en la carpeta de pases de abordar
	timestamp := time.Now().UnixNano()
	savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pases_abordo", fmt.Sprintf("fast_upload_%d_", timestamp))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al guardar el archivo: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    savedPath,
		"message": "Archivo subido correctamente",
		"success": true,
	})
}

func (ctrl *DescargoDerechoController) GetModalLiquidar(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	utils.Render(c, "descargo/components/modal_liquidar", gin.H{
		"Descargo":   descargo,
		"ActionURL":  fmt.Sprintf("/descargos/derecho/%s/liquidar", id),
		"csrf_token": c.GetString("csrf_token"),
	})
}

func (ctrl *DescargoDerechoController) Liquidar(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var req dtos.LiquidarDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Error bind liquidación: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=MontoInvalido")
		return
	}

	var montosUtil, montosCred, montosDevo []float64
	for i := range req.PasajeIDs {
		mu, _ := strconv.ParseFloat(utils.GetIdx(req.CostosUtilizacion, i), 64)
		mc, _ := strconv.ParseFloat(utils.GetIdx(req.MontosCredito, i), 64)
		md, _ := strconv.ParseFloat(utils.GetIdx(req.MontosDevolucion, i), 64)
		montosUtil = append(montosUtil, mu)
		montosCred = append(montosCred, mc)
		montosDevo = append(montosDevo, md)
	}

	if err := ctrl.descargoService.Liquidate(c.Request.Context(), id, req.PasajeIDs, montosUtil, montosCred, montosDevo, authUser.ID); err != nil {
		log.Printf("Error liquidando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorLiquidacion")
		return
	}

	if c.GetHeader("HX-Request") != "" {
		c.Header("HX-Redirect", "/descargos/derecho/"+id)
		c.Status(http.StatusOK)
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) GetModalPago(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	utils.Render(c, "descargo/components/modal_reportar_pago", gin.H{
		"Descargo":   descargo,
		"csrf_token": c.GetString("csrf_token"),
	})
}

func (ctrl *DescargoDerechoController) ReportarPago(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	file, err := c.FormFile("comprobante")
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ArchivoRequerido")
		return
	}

	// Guardar el archivo de comprobante
	timestamp := time.Now().UnixNano()
	savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pagos", fmt.Sprintf("pago_%d_", timestamp))
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorArchivo")
		return
	}

	if err := ctrl.descargoService.ReportPayment(c.Request.Context(), id, savedPath, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorProceso")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) Finalize(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.Finalize(c.Request.Context(), id, authUser.ID); err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorCierre")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RevertLiquidation(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertLiquidation(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error revirtiendo liquidación: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorReversion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RevertPayment(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertPayment(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error revirtiendo pago: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorReversionPago")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RevertFinalization(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if err := ctrl.descargoService.RevertFinalization(c.Request.Context(), id, authUser.ID); err != nil {
		log.Printf("Error revirtiendo finalización: %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorReversion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}
