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

	"github.com/gin-gonic/gin"
)

type DescargoDerechoController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
	destinoService   *services.DestinoService
	reportService    *services.ReportService
	peopleService    *services.PeopleService
	aerolineaService *services.AerolineaService
}

func NewDescargoDerechoController(
	descargoService *services.DescargoService,
	solicitudService *services.SolicitudService,
	destinoService *services.DestinoService,
	reportService *services.ReportService,
	peopleService *services.PeopleService,
	aerolineaService *services.AerolineaService,
	usuarioService *services.UsuarioService,
) *DescargoDerechoController {
	return &DescargoDerechoController{
		descargoService:  descargoService,
		solicitudService: solicitudService,
		destinoService:   destinoService,
		reportService:    reportService,
		peopleService:    peopleService,
		aerolineaService: aerolineaService,
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

	existe, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/derecho/"+existe.ID)
		return
	}

	type ConnectionView struct {
		Ruta   string
		Fecha  string
		Boleto string
		Index  string
	}

	type TicketView struct {
		Boleto string
		Scales []ConnectionView
	}

	pasajesOriginales := make(map[string][]TicketView)
	pasajesReprogramados := make(map[string][]TicketView)

	for _, item := range solicitud.Items {
		tipo := string(item.Tipo)
		for _, p := range item.Pasajes {
			st := p.GetEstadoCodigo()
			if st != "EMITIDO" {
				continue
			}

			targetMap := pasajesOriginales
			prefix := "io"
			if p.PasajeAnteriorID != nil {
				targetMap = pasajesReprogramados
				prefix = "ir"
			}

			ticket := TicketView{
				Boleto: p.NumeroBoleto,
			}

			routes := utils.SplitRoute(p.Ruta)
			for j, r := range routes {
				ticket.Scales = append(ticket.Scales, ConnectionView{
					Ruta:   r,
					Fecha:  p.FechaVuelo.Format("2006-01-02"),
					Boleto: p.NumeroBoleto,
					Index:  fmt.Sprintf("%s_%s_%d", prefix, item.ID, j), // Fixed type to %s for item.ID
				})
			}
			targetMap[tipo] = append(targetMap[tipo], ticket)
		}
	}

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	aerolineaNombre := solicitud.AerolineaSugerida
	if solicitud.AerolineaSugerida != "" {
		if aereolinea, err := ctrl.aerolineaService.GetByID(c.Request.Context(), solicitud.AerolineaSugerida); err == nil {
			if aereolinea.Sigla != "" {
				aerolineaNombre = aereolinea.Sigla
			} else {
				aerolineaNombre = aereolinea.Nombre
			}
		}
	}

	utils.Render(c, "descargo/derecho/create", gin.H{
		"Title":                "Nuevo Descargo (Derecho)",
		"Solicitud":            solicitud,
		"AerolineaNombre":      aerolineaNombre,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"Destinos":             destinos,
	})
}

func (ctrl *DescargoDerechoController) Store(c *gin.Context) {
	var req dtos.CreateDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes?error=DatosInvalidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	// Procesar Archivos de Pases a Bordo
	indices := c.PostFormArray("itin_index[]")
	var archivoPaths []string
	for _, idx := range indices {
		path := ""
		if file, err := c.FormFile("itin_archivo_" + idx); err == nil {
			savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_descargo_"+idx+"_")
			if err == nil {
				path = savedPath
			}
		}
		archivoPaths = append(archivoPaths, path)
	}

	// Procesar Anexos
	var anexoPaths []string
	form, _ := c.MultipartForm()
	files := form.File["anexos[]"]
	for _, fileHeader := range files {
		savedPath, err := utils.SaveUploadedFile(c, fileHeader, "uploads/anexos", "anexo_"+req.SolicitudID+"_")
		if err == nil {
			anexoPaths = append(anexoPaths, savedPath)
		}
	}

	descargo, err := ctrl.descargoService.Create(c.Request.Context(), req, authUser.ID, archivoPaths, anexoPaths)
	if err != nil {
		log.Printf("Error creando descargo derecho: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

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
		key := fmt.Sprintf("%s_%s_%s", item.Tipo, item.Ruta, item.Boleto)
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

				routes := utils.SplitRoute(p.Ruta)
				for _, r := range routes {
					key := fmt.Sprintf("%s_%s_%s", tipoTarget, r, p.NumeroBoleto)
					if !existingKeys[key] {
						tVuelo := p.FechaVuelo
						newItem := models.DetalleItinerarioDescargo{
							Tipo:         models.TipoDetalleItinerario(tipoTarget),
							Ruta:         r,
							Fecha:        &tVuelo,
							Boleto:       p.NumeroBoleto,
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
		Boleto         string
		Detalles       []models.DetalleItinerarioDescargo
		EsDevolucion   bool
		EsModificacion bool
	}

	ticketsMap := make(map[string]*TicketGroup)
	var ticketsOrder []string

	for _, d := range detalles {
		boletoKey := d.Boleto
		if boletoKey == "" {
			boletoKey = "SIN_BOLETO"
		}
		if _, ok := ticketsMap[boletoKey]; !ok {
			ticketsMap[boletoKey] = &TicketGroup{Boleto: d.Boleto}
			ticketsOrder = append(ticketsOrder, boletoKey)
		}

		// If any scale in the ticket is marked, the whole ticket group is marked
		if d.EsDevolucion {
			ticketsMap[boletoKey].EsDevolucion = true
		}
		if d.EsModificacion {
			ticketsMap[boletoKey].EsModificacion = true
		}

		ticketsMap[boletoKey].Detalles = append(ticketsMap[boletoKey].Detalles, d)
	}

	var tickets []TicketGroup
	for _, key := range ticketsOrder {
		tickets = append(tickets, *ticketsMap[key])
	}

	utils.Render(c, "descargo/derecho/show", gin.H{
		"Title":    "Detalle de Descargo (Derecho)",
		"Descargo": descargo,
		"Detalles": detalles,
		"Tickets":  tickets,
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

	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	existingKeys := make(map[string]bool)
	for _, item := range descargo.DetallesItinerario {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
		// Unique key to avoid duplicates
		key := fmt.Sprintf("%s_%s_%s", item.Tipo, item.Ruta, item.Boleto)
		existingKeys[key] = true
	}

	// Dynamic Sync: Check if new pasajes were issued after descargo creation
	if descargo.Solicitud != nil {
		for _, sItem := range descargo.Solicitud.Items {
			tipoBase := string(sItem.Tipo) // IDA or VUELTA
			for _, p := range sItem.Pasajes {
				st := p.GetEstadoCodigo()
				if st != "EMITIDO" && st != "USADO" {
					continue
				}

				tipoTarget := tipoBase + "_ORIGINAL"
				if p.PasajeAnteriorID != nil {
					tipoTarget = tipoBase + "_REPRO"
				}

				routes := utils.SplitRoute(p.Ruta)
				for _, r := range routes {
					key := fmt.Sprintf("%s_%s_%s", tipoTarget, r, p.NumeroBoleto)
					if !existingKeys[key] {
						tVuelo := p.FechaVuelo
						newItem := models.DetalleItinerarioDescargo{
							Tipo:         models.TipoDetalleItinerario(tipoTarget),
							Ruta:         r,
							Fecha:        &tVuelo,
							Boleto:       p.NumeroBoleto,
							EsDevolucion: false,
						}
						itemsByType[tipoTarget] = append(itemsByType[tipoTarget], newItem)
						existingKeys[key] = true
					}
				}
			}
		}
	}

	// Group by ticket structure for the template
	type ConnectionView struct {
		Ruta    string
		Fecha   string
		Boleto  string
		Index   string
		Pase    string
		Archivo string
	}

	type TicketView struct {
		Boleto         string
		Scales         []ConnectionView
		EsDevolucion   bool
		EsModificacion bool
	}

	pasajesOriginales := make(map[string][]TicketView)
	pasajesReprogramados := make(map[string][]TicketView)

	for tipo, items := range itemsByType {
		ticketMap := make(map[string]*TicketView)
		var orderedTickets []*TicketView

		for i, item := range items {
			key := item.Boleto
			if key == "" {
				key = fmt.Sprintf("SN-%v-%d", item.Tipo, i)
			}

			if _, ok := ticketMap[key]; !ok {
				t := &TicketView{
					Boleto:         item.Boleto,
					EsDevolucion:   item.EsDevolucion,
					EsModificacion: item.EsModificacion,
				}
				ticketMap[key] = t
				orderedTickets = append(orderedTickets, t)
			}

			p := "io"
			if strings.HasSuffix(tipo, "REPRO") {
				p = "ir"
			}
			if strings.HasPrefix(tipo, "VUELTA") {
				p = "vo"
			}
			if strings.HasPrefix(tipo, "VUELTA") && strings.HasSuffix(tipo, "REPRO") {
				p = "vr"
			}

			idx := fmt.Sprintf("%s_%s_%d", p, id, i)

			ticketMap[key].Scales = append(ticketMap[key].Scales, ConnectionView{
				Ruta:    item.Ruta,
				Fecha:   item.Fecha.Format("2006-01-02"),
				Boleto:  item.Boleto,
				Index:   idx,
				Pase:    item.NumeroPaseAbordo,
				Archivo: item.ArchivoPaseAbordo,
			})
		}

		deref := make([]TicketView, len(orderedTickets))
		for i, t := range orderedTickets {
			deref[i] = *t
		}

		if strings.HasPrefix(tipo, "IDA") {
			if strings.HasSuffix(tipo, "ORIGINAL") {
				pasajesOriginales["IDA"] = append(pasajesOriginales["IDA"], deref...)
			} else {
				pasajesReprogramados["IDA"] = append(pasajesReprogramados["IDA"], deref...)
			}
		} else {
			if strings.HasSuffix(tipo, "ORIGINAL") {
				pasajesOriginales["VUELTA"] = append(pasajesOriginales["VUELTA"], deref...)
			} else {
				pasajesReprogramados["VUELTA"] = append(pasajesReprogramados["VUELTA"], deref...)
			}
		}
	}

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	aerolineaNombre := descargo.Solicitud.AerolineaSugerida
	if descargo.Solicitud.AerolineaSugerida != "" {
		if aereolinea, err := ctrl.aerolineaService.GetByID(c.Request.Context(), descargo.Solicitud.AerolineaSugerida); err == nil {
			if aereolinea.Sigla != "" {
				aerolineaNombre = aereolinea.Sigla
			} else {
				aerolineaNombre = aereolinea.Nombre
			}
		}
	}

	utils.Render(c, "descargo/derecho/edit", gin.H{
		"Title":                "Editar Descargo (Derecho)",
		"Descargo":             descargo,
		"Solicitud":            descargo.Solicitud,
		"AerolineaNombre":      aerolineaNombre,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"Destinos":             destinos,
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

	indices := c.PostFormArray("itin_index[]")
	var archivoPaths []string
	for _, idx := range indices {
		path := c.PostForm("itin_archivo_existente_" + idx)
		if file, err := c.FormFile("itin_archivo_" + idx); err == nil {
			savedPath, err := utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_descargo_"+idx+"_")
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

	if err := ctrl.descargoService.UpdateFull(c.Request.Context(), id, req, authUser.ID, archivoPaths, anexoPaths); err != nil {
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

	c.Header("Content-Disposition", "inline; filename=PV5_"+descargo.Codigo+".pdf")
	c.Header("Content-Type", "application/pdf")
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

	if err := ctrl.descargoService.Reject(c.Request.Context(), id, authUser.ID); err != nil {
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

	isPDF := strings.HasSuffix(strings.ToLower(path), ".pdf")

	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":                "Previsualización de Documento",
		"FilePath":             fullPath,
		"IsPDF":                isPDF,
		"InfoRuta":             c.Query("ruta"),
		"InfoFecha":            c.Query("fecha"),
		"InfoBoleto":           c.Query("boleto"),
		"InfoVuelo":            c.Query("vuelo"),
		"InfoTramoRegistrado":  c.Query("info_tramo_registrado"),
		"InfoFechaRegistrada":  c.Query("info_fecha_registrada"),
		"InfoBoletoRegistrado": c.Query("info_boleto_registrado"),
		"InfoPaseRegistrado":   c.Query("info_pase_registrado"),
	})
}
