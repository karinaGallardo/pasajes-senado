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
	"strings"

	"github.com/gin-gonic/gin"
)

type DescargoOficialController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
	destinoService   *services.DestinoService
	reportService    *services.ReportService
	peopleService    *services.PeopleService
	configService    *services.ConfiguracionService
}

func NewDescargoOficialController(
	descargoService *services.DescargoService,
	solicitudService *services.SolicitudService,
	destinoService *services.DestinoService,
	reportService *services.ReportService,
	peopleService *services.PeopleService,
	configService *services.ConfiguracionService,
) *DescargoOficialController {
	return &DescargoOficialController{
		descargoService:  descargoService,
		solicitudService: solicitudService,
		destinoService:   destinoService,
		reportService:    reportService,
		peopleService:    peopleService,
		configService:    configService,
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

	existe, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+existe.ID)
		return
	}

	type ConnectionView struct {
		Ruta   string
		Fecha  string
		Boleto string
	}

	pasajesOriginales := make(map[string][]ConnectionView)
	pasajesReprogramados := make(map[string][]ConnectionView)

	for _, item := range solicitud.Items {
		tipo := string(item.Tipo)
		orig := item.GetPasajeOriginal()
		if orig != nil {
			routes := utils.SplitRoute(orig.Ruta)
			for _, r := range routes {
				pasajesOriginales[tipo] = append(pasajesOriginales[tipo], ConnectionView{
					Ruta:   r,
					Fecha:  orig.FechaVuelo.Format("2006-01-02"),
					Boleto: orig.NumeroBoleto,
				})
			}
		}

		repro := item.GetPasajeReprogramado()
		if repro != nil {
			routes := utils.SplitRoute(repro.Ruta)
			for _, r := range routes {
				pasajesReprogramados[tipo] = append(pasajesReprogramados[tipo], ConnectionView{
					Ruta:   r,
					Fecha:  repro.FechaVuelo.Format("2006-01-02"),
					Boleto: repro.NumeroBoleto,
				})
			}
		}
	}

	hasGastosRep := false
	for _, v := range solicitud.Viaticos {
		if v.TieneGastosRep && (v.MontoGastosRep > 0 || v.MontoLiquidoGastos > 0) {
			hasGastosRep = true
			break
		}
	}

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	utils.Render(c, "descargo/oficial/create", gin.H{
		"Title":                "Nuevo Descargo (Oficial)",
		"Solicitud":            solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"HasGastosRep":         hasGastosRep,
		"Destinos":             destinos,
		"ZeroFloat":            float64(0),
		"BancoCuenta":          bancoCuenta,
		"BancoNombre":          bancoNombre,
	})
}

func (ctrl *DescargoOficialController) Store(c *gin.Context) {
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
		log.Printf("Error creando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+descargo.ID+"/editar")
}

func (ctrl *DescargoOficialController) Show(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	// Dynamic Sync for Show View (similar to Edit)
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
						detalles = append(detalles, newItem)
						itemsByType[tipoTarget] = append(itemsByType[tipoTarget], newItem)
						existingKeys[key] = true
					}
				}
			}
		}
	}

	// Group by Ticket for the Template
	type TicketGroup struct {
		Boleto   string
		Detalles []models.DetalleItinerarioDescargo
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
		ticketsMap[boletoKey].Detalles = append(ticketsMap[boletoKey].Detalles, d)
	}

	bancoCuenta := ctrl.configService.GetValue(c.Request.Context(), "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := ctrl.configService.GetValue(c.Request.Context(), "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	var tickets []TicketGroup
	for _, key := range ticketsOrder {
		tickets = append(tickets, *ticketsMap[key])
	}

	utils.Render(c, "descargo/oficial/show", gin.H{
		"Title":       "Detalle de Descargo (Oficial)",
		"Descargo":    descargo,
		"Detalles":    detalles,
		"Tickets":     tickets,
		"BancoCuenta": bancoCuenta,
		"BancoNombre": bancoNombre,
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

	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	existingKeys := make(map[string]bool)
	for _, item := range descargo.DetallesItinerario {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
		key := fmt.Sprintf("%s_%s_%s", item.Tipo, item.Ruta, item.Boleto)
		existingKeys[key] = true
	}

	// Dynamic Sync: Check if new pasajes were issued after descargo creation
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
						itemsByType[tipoTarget] = append(itemsByType[tipoTarget], newItem)
						existingKeys[key] = true
					}
				}
			}
		}
	}

	// Group by ticket structure for the template
	type ConnectionView struct {
		Ruta           string
		Fecha          string
		Boleto         string
		Index          string
		Pase           string
		Archivo        string
		EsDevolucion   bool
		EsModificacion bool
	}

	pasajesOriginales := make(map[string][]ConnectionView)
	pasajesReprogramados := make(map[string][]ConnectionView)

	for tipo, items := range itemsByType {
		for i, item := range items {
			prefix := "io" // ida original
			if tipo == "IDA_REPRO" {
				prefix = "ir"
			} else if tipo == "VUELTA_ORIGINAL" {
				prefix = "vo"
			} else if tipo == "VUELTA_REPRO" {
				prefix = "vr"
			}

			view := ConnectionView{
				Ruta:           item.Ruta,
				Fecha:          "",
				Boleto:         item.Boleto,
				Index:          fmt.Sprintf("%s_%d", prefix, i),
				Pase:           item.NumeroPaseAbordo,
				Archivo:        item.ArchivoPaseAbordo,
				EsDevolucion:   item.EsDevolucion,
				EsModificacion: item.EsModificacion,
			}
			if item.Fecha != nil {
				view.Fecha = item.Fecha.Format("2006-01-02")
			}

			if strings.HasSuffix(tipo, "_ORIGINAL") {
				pasajesOriginales[tipo] = append(pasajesOriginales[tipo], view)
			} else {
				pasajesReprogramados[tipo] = append(pasajesReprogramados[tipo], view)
			}
		}
	}

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
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
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"Destinos":             destinos,
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
	pdfReader, err := ctrl.reportService.GeneratePV06Complete(c.Request.Context(), descargo, personaView)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generando PDF")
		return
	}

	c.Header("Content-Disposition", "inline; filename=PV6_"+descargo.Codigo+".pdf")
	c.Header("Content-Type", "application/pdf")
	c.Writer.Write(pdfReader)
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
