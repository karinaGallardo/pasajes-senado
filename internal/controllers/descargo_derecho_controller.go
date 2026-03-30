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

	// Delegamos la transformación de datos al servicio (Lógica de Negocio)
	pasajesOriginales, pasajesReprogramados := ctrl.descargoService.GetItinerarioParaDescargo(c.Request.Context(), solicitud)

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	utils.Render(c, "descargo/derecho/create", gin.H{
		"Title":                "Nuevo Descargo (Derecho)",
		"Solicitud":            solicitud,
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
		CostoPasaje     float64
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

	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	existingKeys := make(map[string]bool)
	for _, item := range descargo.DetallesItinerario {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
		// Unique key to avoid duplicates
		key := fmt.Sprintf("%s_%d_%s", item.Tipo, item.Orden, strings.ToUpper(strings.TrimSpace(item.Boleto)))
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

				segments := p.GetRutaSegments()
				for i := range segments {
					// Clave compuesta estricta: Tipo + Orden + Boleto
					key := fmt.Sprintf("%s_%d_%s", tipoTarget, i, strings.ToUpper(strings.TrimSpace(p.NumeroBoleto)))
					if !existingKeys[key] {
						tVuelo := p.FechaVuelo
						newItem := models.DetalleItinerarioDescargo{
							Tipo:         models.TipoDetalleItinerario(tipoTarget),
							RutaID:       p.RutaID,
							Fecha:        &tVuelo,
							Boleto:       p.NumeroBoleto,
							Orden:        i,
							EsDevolucion: false,
						}
						itemsByType[tipoTarget] = append(itemsByType[tipoTarget], newItem)
						existingKeys[key] = true
					}
				}
			}
		}
	}

	pasajesOriginales := make(map[string][]dtos.TicketView)
	pasajesReprogramados := make(map[string][]dtos.TicketView)

	// Procesamiento ordenado para garantizar que Ida aparezca antes que Vuelta en la vista
	tiposOrdenados := []string{"IDA_ORIGINAL", "IDA_REPRO", "VUELTA_ORIGINAL", "VUELTA_REPRO"}
	for _, tipo := range tiposOrdenados {
		items, ok := itemsByType[tipo]
		if !ok {
			continue
		}
		ticketMap := make(map[string]*dtos.TicketView)
		var orderedTickets []*dtos.TicketView

		// Pre-escaneo para identificar boletos con rutas válidas en este grupo
		itemHasValidRoute := make(map[string]bool)
		for _, itm := range items {
			bKey := itm.Boleto
			if bKey == "" {
				bKey = "SIN_BOLETO"
			}
			if itm.GetRutaDisplay() != "Ruta no especificada" {
				itemHasValidRoute[bKey] = true
			}
		}

		for i, item := range items {
			key := item.Boleto
			if key == "" {
				key = fmt.Sprintf("SN-%v-%d", item.Tipo, i)
			}

			if _, ok := ticketMap[key]; !ok {
				t := &dtos.TicketView{
					Boleto:          item.Boleto,
					EsDevolucion:    item.EsDevolucion,
					EsModificacion:  item.EsModificacion,
					MontoDevolucion: item.MontoDevolucion,
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

			// Find original cost
			costoPasaje := 0.0
			if descargo.Solicitud != nil {
				for _, sitem := range descargo.Solicitud.Items {
					for _, pas := range sitem.Pasajes {
						if pas.NumeroBoleto == item.Boleto && item.Boleto != "" {
							costoPasaje = pas.Costo
							break
						}
					}
				}
			}

			idx := fmt.Sprintf("%s_%s_%d", p, id, i)

			// Evitar duplicados por ruta en el mismo boleto USANDO CLAVE COMPUESTA
			isDuplicate := false
			// Si el ticketMap ya tiene este boleto, verificar si ya metimos este tramo (Orden)
			if t, ok := ticketMap[key]; ok {
				for _, sc := range t.Scales {
					if sc.Orden == item.Orden {
						isDuplicate = true
						break
					}
				}
			}

			// Si este boleto ya tiene una ruta válida, ignorar cualquier "Ruta no especificada"
			if item.GetRutaDisplay() == "Ruta no especificada" && itemHasValidRoute[item.Boleto] {
				isDuplicate = true
			}

			if !isDuplicate {
				dateStr := ""
				if item.Fecha != nil {
					dateStr = item.Fecha.Format("2006-01-02")
				}

				ticketMap[key].Scales = append(ticketMap[key].Scales, dtos.ConnectionView{
					ID:              item.ID,
					Ruta:            item.GetRutaDisplay(),
					RutaID:          utils.DerefString(item.RutaID),
					Fecha:           dateStr,
					Boleto:          item.Boleto,
					Index:           idx,
					Pase:            item.NumeroPaseAbordo,
					Archivo:         item.ArchivoPaseAbordo,
					EsDevolucion:    item.EsDevolucion,
					EsModificacion:  item.EsModificacion,
					MontoDevolucion: item.MontoDevolucion,
					CostoPasaje:     costoPasaje,
					Orden:           item.Orden,
				})
			}
		}

		// Scales within a ticket are already sorted by the original insertion order of the slices.
		// No need for grouping metadata (IsFirstScale, TotalScales) in the atomic model.

		deref := make([]dtos.TicketView, len(orderedTickets))
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

	utils.Render(c, "descargo/derecho/edit", gin.H{
		"Title":                "Editar Descargo (Derecho)",
		"Descargo":             descargo,
		"Solicitud":            descargo.Solicitud,
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

	// Captura manual de todos los arrays para asegurar sincronización perfecta entre ellos
	req.ItinTipo = c.PostFormArray("itin_tipo[]")
	req.ItinID = c.PostFormArray("itin_id[]")
	req.ItinRutaID = c.PostFormArray("itin_ruta_id[]")
	req.ItinFecha = c.PostFormArray("itin_fecha[]")
	req.ItinBoleto = c.PostFormArray("itin_boleto[]")
	req.ItinPaseNumero = c.PostFormArray("itin_pase_numero[]")
	req.ItinIndex = c.PostFormArray("itin_index[]")
	req.ItinOrden = c.PostFormArray("itin_orden[]")
	req.ItinDevolucion = c.PostFormArray("itin_devolucion[]")
	req.ItinModificacion = c.PostFormArray("itin_modificacion[]")
	req.ItinMontoDevolucion = c.PostFormArray("itin_monto_devo[]")

	indices := req.ItinIndex
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
	index := fmt.Sprintf("new_%d", time.Now().UnixNano())

	c.HTML(http.StatusOK, "descargo/components/escala_fila_derecho", gin.H{
		"Tipo":            tipo,
		"Index":           index,
		"EsDevolucion":    false,
		"EsModificacion":  false,
		"Ruta":            "",
		"RutaID":          "",
		"Fecha":           "",
		"Boleto":          "",
		"Pase":            "",
		"MontoDevolucion": 0.0,
		"CostoPasaje":     0.0,
		"Archivo":         "",
		"ReadOnlyCheck":   false,
		"ReadOnlyRuta":    false,
		"CanDelete":       true,
		"IsFirstScale":    true,
		"TotalScales":     1,
		"Orden":           0,
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
