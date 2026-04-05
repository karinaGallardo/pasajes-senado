package controllers

import (
	"fmt"
	"log"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
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
) *DescargoDerechoController {
	return &DescargoDerechoController{
		descargoService:        descargoService,
		descargoDerechoService: descargoDerechoService,
		solicitudService:       solicitudService,
		destinoService:         destinoService,
		reportService:          reportService,
		peopleService:          peopleService,
		aerolineaService:       aerolineaService,
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

	utils.Render(c, "descargo/derecho/show", gin.H{
		"Title":      "Detalle de Descargo (Derecho)",
		"Descargo":   data.Descargo,
		"Ida":        data.Ida,
		"Vuelta":     data.Vuelta,
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

func (ctrl *DescargoDerechoController) PreviewFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.String(http.StatusBadRequest, "Ruta de archivo requerida")
		return
	}

	fullPath := path
	if !strings.HasPrefix(path, "http") && !strings.HasPrefix(path, "/") {
		fullPath = "/" + path
	}

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
		"FilePath":              fullPath,
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
		"Tramo": dtos.TramoView{
			ID:              index,
			Tipo:            tipo,
			SolicitudItemID: solicitudItemID,
			EsDevolucion:    false,
			EsModificacion:  false,
			Ruta:            dtos.RutaView{},
			RutaID:          "",
			Fecha:           "",
			Billete:         "",
			Pase:            "",
			MontoDevolucion: 0.0,
			Moneda:          "Bs.",
			Archivo:         "",
			PasajeID:        "",
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
