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

func (ctrl *DescargoOficialController) Create(c *gin.Context) {
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

	pasajesOriginales, pasajesReprogramados := ctrl.descargoOficialService.PrepareItinerarioOficial(descargo)

	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	utils.Render(c, "descargo/oficial/show", gin.H{
		"Title":                "Detalle de Descargo (Oficial)",
		"Descargo":             descargo,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"BancoCuenta":          bancoCuenta,
		"BancoNombre":          bancoNombre,
	})
}

func (ctrl *DescargoOficialController) Edit(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
		return
	}

	// Sincronización proactiva de pasajes emitidos después de la creación
	if descargo.Solicitud != nil {
		if err := ctrl.descargoOficialService.SyncItineraryFromSolicitud(c.Request.Context(), descargo, descargo.Solicitud); err == nil {
			// Recargar para popular Preloads de los nuevos items sincronizados
			descargo, _ = ctrl.descargoService.GetByID(c.Request.Context(), id)
		} else {
			log.Printf("Error sincronizando itinerario oficial en edición: %v", err)
		}
	}

	pasajesOriginales, pasajesReprogramados := ctrl.descargoOficialService.PrepareItinerarioOficial(descargo)

	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	utils.Render(c, "descargo/oficial/edit", gin.H{
		"Title":                "Editar Descargo (Oficial)",
		"Descargo":             descargo,
		"Solicitud":            descargo.Solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"BancoCuenta":          bancoCuenta,
		"BancoNombre":          bancoNombre,
	})
}

func (ctrl *DescargoOficialController) Update(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"?error=EstadoNoPermitido")
		return
	}

	var req dtos.CreateDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id+"/editar?error=DatosInvalidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	// RE-BINDING MANUAL: Gin sometimes fails to bind parallel arrays in complex multipart/form-data
	req.ItinTipo = c.PostFormArray("itin_tipo[]")
	req.ItinID = c.PostFormArray("itin_id[]")
	req.ItinRutaID = c.PostFormArray("itin_ruta_id[]")
	req.ItinFecha = c.PostFormArray("itin_fecha[]")
	req.ItinBoleto = c.PostFormArray("itin_boleto[]")
	req.ItinPaseNumero = c.PostFormArray("itin_pase_numero[]")
	req.ItinOrden = c.PostFormArray("itin_orden[]")
	req.ItinDevolucion = c.PostFormArray("itin_devolucion[]")
	req.ItinModificacion = c.PostFormArray("itin_modificacion[]")
	req.ItinMontoDevolucion = c.PostFormArray("itin_monto_devolucion[]")
	req.ItinMoneda = c.PostFormArray("itin_moneda[]")
	req.ItinPasajeID = c.PostFormArray("itin_pasaje_id[]")
	req.ItinSolicitudItemID = c.PostFormArray("itin_solicitud_item_id[]")

	var archivoPaths []string
	for _, idRow := range req.ItinID {
		path := c.PostForm("itin_archivo_existente_" + idRow)
		if file, err := c.FormFile("itin_archivo_" + idRow); err == nil {
			savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_descargo_"+idRow+"_")
			if err == nil {
				path = savedPath
			}
		}
		archivoPaths = append(archivoPaths, path)
	}

	var anexoPaths []string
	form, _ := c.MultipartForm()
	newAnexos := form.File["anexos[]"]
	existentes := c.PostFormArray("anexos_existentes[]")
	anexoPaths = append(anexoPaths, existentes...)

	for _, fileHeader := range newAnexos {
		savedPath, err := utils.SaveUploadedFile(c, fileHeader, "uploads/anexos", "anexo_edit_"+id+"_")
		if err == nil {
			anexoPaths = append(anexoPaths, savedPath)
		}
	}

	if err := ctrl.descargoOficialService.UpdateOficial(c.Request.Context(), id, req, authUser.ID, archivoPaths, anexoPaths); err != nil {
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

	c.HTML(http.StatusOK, "descargo/components/escala_fila_oficial", gin.H{
		"Tipo": tipo,
		"Scale": dtos.ConnectionView{
			ID:              index,
			SolicitudItemID: solicitudItemID,
			EsModificacion:  false,
			Ruta:            dtos.RutaView{},
			RutaID:          "",
			Fecha:           "",
			Boleto:          "",
			Pase:            "",
			Archivo:         "",
			Orden:           0,
			PasajeID:        "",
		},
	})
}
