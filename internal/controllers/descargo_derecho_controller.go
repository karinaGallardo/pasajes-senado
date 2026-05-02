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

	csrfToken := c.Value("csrf_token")

	utils.Render(c, "descargo/derecho/edit", gin.H{
		"Title":      "Editar Descargo",
		"Descargo":   data.Descargo,
		"Solicitud":  data.Solicitud,
		"Ida":        data.Ida,
		"Vuelta":     data.Vuelta,
		"csrf_token": csrfToken,
		"LinkBase":   "/descargos/derecho",
	})
}

func (ctrl *DescargoDerechoController) Completar(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	data, err := ctrl.descargoDerechoService.GetEditData(c.Request.Context(), id)
	if err != nil {
		// Even if OPEN_TICKET is technically considered "No Editable" by default rules,
		// GetEditData already allows OPEN_TICKET if IsEditable is updated.
		// Wait, let's verify if GetEditData allows OPEN_TICKET.
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=NoEditable")
		return
	}

	data.Descargo.HydratePermissions(authUser)
	csrfToken := c.Value("csrf_token")

	utils.Render(c, "descargo/derecho/completar", gin.H{
		"Title":      "Completar Pasajes (Open Ticket)",
		"Descargo":   data.Descargo,
		"Solicitud":  data.Solicitud,
		"Ida":        data.Ida,
		"Vuelta":     data.Vuelta,
		"csrf_token": csrfToken,
		"LinkBase":   "/descargos/derecho",
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

	// 4. Comprobantes de Pago (Per Pasaje)
	boletasPaths := utils.ExtractPasajeBoletas(c, req.LiquidacionPasajeID)

	if err := ctrl.descargoDerechoService.UpdateDerecho(c.Request.Context(), id, req, authUser.ID, pasesAbordoPaths, boletasPaths); err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar?error=ErrorActualizacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar")
}

func (ctrl *DescargoDerechoController) UpdateReutilizacion(c *gin.Context) {
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

	// Permisos: Si está en OPEN_TICKET, se permite completar
	descargo.HydratePermissions(authUser)
	if !descargo.Permissions.CanCompleteOpenTicket {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=EstadoNoPermitido")
		return
	}

	var req dtos.CreateDescargoRequest
	if err := req.Bind(c); err != nil {
		log.Printf("[ERROR] Bind error en Reutilización (ID: %s): %v", id, err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/completar?error=DatosInvalidos")
		return
	}

	// Extraer pases a bordo de los tramos (incluyendo los nuevos REUT)
	pasesAbordoPaths := utils.ExtractDescargoFiles(c, req.TramoID)

	// Comprobantes de Pago (Per Pasaje)
	boletasPaths := utils.ExtractPasajeBoletas(c, req.LiquidacionPasajeID)

	// Asegurar que el estado se mantiene en OPEN_TICKET durante este flujo
	descargo.Estado = models.EstadoDescargoOpenTicket
	if err := ctrl.descargoDerechoService.UpdateDerecho(c.Request.Context(), id, req, authUser.ID, pasesAbordoPaths, boletasPaths); err != nil {
		log.Printf("[ERROR] Error en UpdateDerecho (Reutilización): %v", err)
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/completar?error=ErrorActualizacion")
		return
	}

	// Redirigir al MISMO FORMULARIO (Completar) para permitir seguir cargando tramos
	c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/completar?success=TramosActualizados")
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

func (ctrl *DescargoDerechoController) PrintOpenTicket(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	pdf := ctrl.reportService.GeneratePV05OpenTicket(c.Request.Context(), descargo)
	if pdf.Err() {
		c.String(http.StatusInternalServerError, "Error generando PDF de Open Ticket")
		return
	}

	disposition := "inline"
	if utils.IsMobileBrowser(c) {
		disposition = "attachment"
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"OPEN-TICKET-%s.pdf\"", disposition, descargo.ID))

	if err := pdf.Output(c.Writer); err != nil {
		log.Printf("Error enviando PDF OT: %v", err)
	}
}

func (ctrl *DescargoDerechoController) Preview(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    "Previsualización Formulario PV-05",
		"FilePath": fmt.Sprintf("/descargos/derecho/%s/imprimir", id),
		"IsPDF":    true,
	})
}

func (ctrl *DescargoDerechoController) PreviewOT(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    "Previsualización Reporte de Reutilización (OT)",
		"FilePath": fmt.Sprintf("/descargos/derecho/%s/imprimir-ot", id),
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

func (ctrl *DescargoDerechoController) Reject(ctx *gin.Context) {
	id := ctx.Param("id")
	authUser := appcontext.AuthUser(ctx)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		ctx.Redirect(http.StatusFound, "/auth/login")
		return
	}

	observaciones := ctx.PostForm("observaciones")

	descargo, _ := ctrl.descargoService.GetByID(ctx.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanReject {
			ctx.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=SinPermisoRechazo")
			return
		}
	}

	if err := ctrl.descargoService.Reject(ctx.Request.Context(), id, authUser.ID, observaciones); err != nil {
		log.Printf("Error rechazando descargo derecho: %v", err)
		ctx.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=ErrorRechazo")
		return
	}

	ctx.Redirect(http.StatusFound, "/descargos/derecho/"+id)
}

func (ctrl *DescargoDerechoController) RevertApproval(ctx *gin.Context) {
	id := ctx.Param("id")
	authUser := appcontext.AuthUser(ctx)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		ctx.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}

	descargo, _ := ctrl.descargoService.GetByID(ctx.Request.Context(), id)
	if descargo != nil {
		descargo.HydratePermissions(authUser)
		if !descargo.Permissions.CanRevert {
			ctx.String(http.StatusForbidden, "No tiene permisos para revertir este descargo")
			return
		}
	}

	if err := ctrl.descargoService.RevertToDraft(ctx.Request.Context(), id, authUser.ID); err != nil {
		ctx.String(http.StatusInternalServerError, "Error revirtiendo aprobación: "+err.Error())
		return
	}

	ctx.Redirect(http.StatusFound, "/descargos/derecho/"+id)
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
	isReutilizacion := c.Query("is_reutilizacion") == "true"
	pasajeID := c.Query("pasaje_id")
	billete := c.Query("billete")

	index := fmt.Sprintf("new_%d", time.Now().UnixNano())

	if isReutilizacion {
		billete = ""
	}

	tramo := models.DescargoTramo{
		BaseModel:       models.BaseModel{ID: index},
		Tipo:            models.TipoDescargoTramo(tipo),
		SolicitudItemID: &solicitudItemID,
		Billete:         billete,
		EsOpenTicket:    false,
		EsModificacion:  false,
	}

	if pasajeID != "" {
		tramo.PasajeID = &pasajeID
	}

	templateName := "descargo/components/tramo_fila_derecho"
	if isReutilizacion {
		templateName = "descargo/components/tramo_fila_reutilizado"
	}

	c.HTML(http.StatusOK, templateName, gin.H{
		"Tipo":            tipo,
		"Tramo":           tramo,
		"IsReutilizacion": isReutilizacion,
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
