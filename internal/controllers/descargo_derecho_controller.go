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
	"sort"
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

func (ctrl *DescargoDerechoController) Create(c *gin.Context) {
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
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	// Dynamic Sync for Show View (similar to Edit)
	// We want to show even items that haven't been formally added yet but are EMITIDO
	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	existingKeys := make(map[string]bool)
	detalles := descargo.DetallesItinerario

	for _, item := range detalles {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
		key := fmt.Sprintf("%s_%d_%s", item.Tipo, item.Orden, strings.ToUpper(strings.TrimSpace(item.Boleto)))
		existingKeys[key] = true
	}

	if descargo.Solicitud != nil {
		for _, sItem := range descargo.Solicitud.Items {
			tipoBase := string(sItem.Tipo)
			for _, p := range sItem.Pasajes {
				st := p.GetEstadoCodigo()
				if st != "EMITIDO" && st != "USADO" {
					continue
				}

				tipoTarget := tipoBase + "_ORIGINAL"
				if p.PasajeAnteriorID != nil {
					tipoTarget = tipoBase + "_REPRO"
				}

				segments := p.GetRutaSegments()
				for i := range segments {
					key := fmt.Sprintf("%s_%d_%s", tipoTarget, i, strings.ToUpper(strings.TrimSpace(p.NumeroBoleto)))
					if !existingKeys[key] {
						tVuelo := p.FechaVuelo
						newItem := models.DetalleItinerarioDescargo{
							Tipo:         models.TipoDetalleItinerario(tipoTarget),
							RutaID:       p.RutaID,
							RutaPasaje:   p.RutaPasaje,
							Fecha:        &tVuelo,
							Boleto:       p.NumeroBoleto,
							Orden:        i,
							EsDevolucion: false,
						}
						// Append to both the general list and the map
						detalles = append(detalles, newItem)
						itemsByType[tipoTarget] = append(itemsByType[tipoTarget], newItem)
						existingKeys[key] = true
					}
				}
			}
		}
	}

	type TicketGroup struct {
		Boleto          string
		Detalles        []models.DetalleItinerarioDescargo
		EsDevolucion    bool
		EsModificacion  bool
		MontoDevolucion float64
	}

	ticketsMap := make(map[string]*TicketGroup)
	var ticketsOrder []string

	// Pre-escaneo para identificar boletos con rutas válidas
	boletoHasValidRoute := make(map[string]bool)
	for _, d := range detalles {
		bKey := d.Boleto
		if bKey == "" {
			bKey = "SIN_BOLETO"
		}
		if d.GetRutaDisplay() != "Ruta no especificada" {
			boletoHasValidRoute[bKey] = true
		}
	}

	for _, d := range detalles {
		boletoKey := d.Boleto
		if boletoKey == "" {
			boletoKey = "SIN_BOLETO"
		}
		if _, ok := ticketsMap[boletoKey]; !ok {
			ticketsMap[boletoKey] = &TicketGroup{Boleto: d.Boleto}
			ticketsOrder = append(ticketsOrder, boletoKey)
		}

		// Evitar duplicados por ruta en el mismo boleto
		isDuplicate := false
		for _, existing := range ticketsMap[boletoKey].Detalles {
			if existing.Orden == d.Orden && existing.Tipo == d.Tipo {
				isDuplicate = true
				break
			}
		}

		// Si ya sabemos que este boleto tiene una ruta válida, ignorar cualquier "Ruta no especificada"
		if d.GetRutaDisplay() == "Ruta no especificada" && boletoHasValidRoute[boletoKey] {
			isDuplicate = true
		}

		if !isDuplicate {
			ticketsMap[boletoKey].Detalles = append(ticketsMap[boletoKey].Detalles, d)
		}
	}

	var ticketsIda []TicketGroup
	var ticketsVuelta []TicketGroup

	for _, key := range ticketsOrder {
		tg := ticketsMap[key]
		sort.Slice(tg.Detalles, func(i, j int) bool {
			return tg.Detalles[i].Orden < tg.Detalles[j].Orden
		})

		// Determine if it belongs to IDA or VUELTA based on any detail type
		isVuelta := false
		if len(tg.Detalles) > 0 {
			tipo := string(tg.Detalles[0].Tipo)
			if strings.HasPrefix(tipo, "VUELTA") {
				isVuelta = true
			}
		}

		if isVuelta {
			ticketsVuelta = append(ticketsVuelta, *tg)
		} else {
			ticketsIda = append(ticketsIda, *tg)
		}
	}

	utils.Render(c, "descargo/derecho/show", gin.H{
		"Title":         "Detalle de Descargo (Derecho)",
		"Descargo":      descargo,
		"Detalles":      detalles,
		"TicketsIda":    ticketsIda,
		"TicketsVuelta": ticketsVuelta,
	})
}

func (ctrl *DescargoDerechoController) Edit(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id)
		return
	}

	// Sincronización proactiva: Si se emitieron nuevos pasajes después de la creación inicial
	if descargo.Solicitud != nil {
		if err := ctrl.descargoDerechoService.SyncItineraryFromSolicitud(c.Request.Context(), descargo, descargo.Solicitud); err == nil {
			// Recargar el descargo para asegurar que las relaciones GORM se pueblen para los nuevos items sincronizados
			descargo, _ = ctrl.descargoService.GetByID(c.Request.Context(), id)
		} else {
			log.Printf("Error sincronizando itinerario en edición: %v", err)
		}
	}

	pasajesOriginales, pasajesReprogramados := ctrl.descargoDerechoService.PrepareItinerarioDerecho(descargo)

	utils.Render(c, "descargo/derecho/edit", gin.H{
		"Title":                "Editar Descargo (Derecho)",
		"Descargo":             descargo,
		"Solicitud":            descargo.Solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
	})
}

func (ctrl *DescargoDerechoController) Update(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"?error=EstadoNoPermitido")
		return
	}

	var req dtos.CreateDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+id+"/editar?error=DatosInvalidos")
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

	// the file handling still needs careful manual processing due to dynamic naming
	var archivoPaths []string
	for _, idRow := range req.ItinID {
		// Try to get existing file path if no new file is uploaded
		path := c.PostForm("itin_archivo_existente_" + idRow)

		// Check for new file upload for this specific row
		if file, err := c.FormFile("itin_archivo_" + idRow); err == nil {
			savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_descargo_"+idRow+"_")
			if err == nil {
				path = savedPath
			}
		}
		archivoPaths = append(archivoPaths, path)
	}

	if err := ctrl.descargoDerechoService.UpdateDerecho(c.Request.Context(), id, req, authUser.ID, archivoPaths); err != nil {
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
		"Title":                title,
		"FilePath":             fullPath,
		"IsPDF":                isPDF,
		"IsImage":              isImage,
		"InfoRuta":             c.Query("ruta"),
		"InfoFecha":            c.Query("fecha"),
		"InfoBoleto":           c.Query("boleto"),
		"InfoVuelo":            c.Query("vuelo"),
		"InfoTramoRegistrado":  c.Query("info_tramo_registrado"),
		"InfoFechaRegistrada":  c.Query("info_fecha_registrada"),
		"InfoBoletoRegistrado": c.Query("info_boleto_registrado"),
		"InfoPaseRegistrado":   c.Query("info_pase_registrado"),
		"IsMobile":             utils.IsMobileBrowser(c),
	})
}

func (ctrl *DescargoDerechoController) NuevaFila(c *gin.Context) {
	tipo := c.Query("tipo")
	solicitudItemID := c.Query("solicitud_item_id")
	index := fmt.Sprintf("new_%d", time.Now().UnixNano())

	c.HTML(http.StatusOK, "descargo/components/escala_fila_derecho", gin.H{
		"Tipo": tipo,
		"Scale": dtos.ConnectionView{
			ID:              index,
			SolicitudItemID: solicitudItemID,
			EsDevolucion:    false,
			EsModificacion:  false,
			Ruta:            dtos.RutaView{},
			RutaID:          "",
			Fecha:           "",
			Boleto:          "",
			Pase:            "",
			MontoDevolucion: 0.0,
			Moneda:          "Bs.",
			Archivo:         "",
			Orden:           0,
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
