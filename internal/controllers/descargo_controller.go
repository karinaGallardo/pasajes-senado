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

type DescargoController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
	reportService    *services.ReportService
	peopleService    *services.PeopleService
}

func NewDescargoController() *DescargoController {
	return &DescargoController{
		descargoService:  services.NewDescargoService(),
		solicitudService: services.NewSolicitudService(),
		reportService:    services.NewReportService(),
		peopleService:    services.NewPeopleService(),
	}
}

func (ctrl *DescargoController) Index(c *gin.Context) {
	descargos, _ := ctrl.descargoService.GetAll(c.Request.Context())
	utils.Render(c, "descargo/index", gin.H{
		"Title":     "Bandeja de Descargos",
		"Descargos": descargos,
	})
}

func (ctrl *DescargoController) CreateDerecho(c *gin.Context) {
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
		c.Redirect(http.StatusFound, "/descargos/"+existe.ID)
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

	utils.Render(c, "descargo/derecho/create", gin.H{
		"Title":                "Nuevo Descargo",
		"Solicitud":            solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
	})
}

// CreateOficial muestra el formulario para crear descargo de una solicitud oficial (pasajes, viáticos, gastos de representación).
func (ctrl *DescargoController) CreateOficial(c *gin.Context) {
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

	// Solo permitir descargo para solicitudes oficiales (sin cupo por derecho)
	if solicitud.CupoDerechoItemID != nil && *solicitud.CupoDerechoItemID != "" {
		c.Redirect(http.StatusFound, "/solicitudes/derecho/"+solicitudID+"/detalle")
		return
	}

	existe, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/"+existe.ID)
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

	utils.Render(c, "descargo/oficial/create", gin.H{
		"Title":                "Formulario de Descargo PV-06 - Pasajes, Viáticos y Gastos de Representación",
		"Solicitud":            solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"HasGastosRep":         hasGastosRep,
		"ZeroFloat":            float64(0),
	})
}

func (ctrl *DescargoController) Store(c *gin.Context) {
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
		log.Printf("Error creando descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/"+descargo.ID+"/editar")
}

func (ctrl *DescargoController) Show(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error buscando descargo %s: %v", id, err)
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	templateName := "descargo/derecho/show"
	if strings.HasPrefix(descargo.Codigo, "SOF") {
		templateName = "descargo/oficial/show"
	}

	utils.Render(c, templateName, gin.H{
		"Title":    "Detalle de Descargo",
		"Descargo": descargo,
	})
}

func (ctrl *DescargoController) Edit(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != "EN_REVISION" {
		c.Redirect(http.StatusFound, "/descargos/"+id)
		return
	}

	// Categorizar items para la vista
	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	for _, item := range descargo.DetallesItinerario {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
	}

	templateName := "descargo/derecho/edit"
	if strings.HasPrefix(descargo.Codigo, "SOF") {
		templateName = "descargo/oficial/edit"
	}

	utils.Render(c, templateName, gin.H{
		"Title":       "Editar Descargo",
		"Descargo":    descargo,
		"ItemsByType": itemsByType,
	})
}

func (ctrl *DescargoController) Update(c *gin.Context) {
	id := c.Param("id")
	var req dtos.CreateDescargoRequest // Can reuse the same DTO for basic fields
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/descargos/"+id+"/editar?error=DatosInvalidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	// Procesar Archivos de Pases a Bordo (Solo los nuevos)
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

	// Procesar Anexos (Nuevos)
	var anexoPaths []string
	// Aquí podríamos tener 'anexos_existentes[]' pero por ahora si sube nuevos reemplaza o agrega.
	// Vamos a simplificar: si hay nuevos, se agregan a los existentes o se reemplazan?
	// PV-06 suele ser un documento final, así que si edita puede querer subir más.
	form, _ := c.MultipartForm()
	newAnexos := form.File["anexos[]"]

	// Recuperar existentes mandados por el form (si quisiéramos persistencia selectiva)
	existentes := c.PostFormArray("anexos_existentes[]")
	anexoPaths = append(anexoPaths, existentes...)

	for _, fileHeader := range newAnexos {
		savedPath, err := utils.SaveUploadedFile(c, fileHeader, "uploads/anexos", "anexo_edit_"+id+"_")
		if err == nil {
			anexoPaths = append(anexoPaths, savedPath)
		}
	}

	if err := ctrl.descargoService.UpdateFull(c.Request.Context(), id, req, authUser.ID, archivoPaths, anexoPaths); err != nil {
		log.Printf("Error actualizando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/"+id+"/editar?error=ErrorActualizacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos/"+id+"/editar")
}

func (ctrl *DescargoController) Approve(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error aprobando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/"+id)
		return
	}

	descargo.Estado = "APROBADO"
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}
	descargo.UpdatedBy = &authUser.ID

	ctrl.descargoService.Update(c.Request.Context(), descargo)

	if descargo.SolicitudID != "" {
		ctrl.solicitudService.Finalize(c.Request.Context(), descargo.SolicitudID)
	}

	c.Redirect(http.StatusFound, "/descargos/"+id)
}

func (ctrl *DescargoController) DownloadPV5ByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.String(http.StatusBadRequest, "ID de descargo requerido")
		return
	}

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	if descargo.Solicitud == nil {
		c.String(http.StatusNotFound, "Solicitud no vinculada")
		return
	}

	personaView, _ := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), descargo.Solicitud.Usuario.CI)

	var pdfReader []byte
	if strings.HasPrefix(descargo.Codigo, "SOF") {
		pdfReader, err = ctrl.reportService.GeneratePV06Complete(c.Request.Context(), descargo, personaView)
	} else {
		pdfReader, err = ctrl.reportService.GeneratePV05Complete(c.Request.Context(), descargo, personaView)
	}

	if err != nil {
		log.Printf("Error generating complete PDF: %v", err)
		c.String(http.StatusInternalServerError, "Error generando PDF")
		return
	}

	filename := "PV5_Descargo_" + descargo.Codigo + ".pdf"
	if strings.HasPrefix(descargo.Codigo, "SOF") {
		filename = "PV6_Descargo_" + descargo.Codigo + ".pdf"
	}
	c.Header("Content-Disposition", "inline; filename="+filename)
	c.Header("Content-Type", "application/pdf")

	if _, err := c.Writer.Write(pdfReader); err != nil {
		log.Printf("Error writing PDF to response: %v", err)
	}
}
func (ctrl *DescargoController) PreviewPV5(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Descargo no encontrado")
		return
	}

	title := "Previsualización Formulario PV-05"
	if strings.HasPrefix(descargo.Codigo, "SOF") {
		title = "Previsualización Formulario PV-06"
	}

	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    title,
		"FilePath": fmt.Sprintf("/descargos/%s/imprimir-pv5", descargo.ID),
		"IsPDF":    true,
	})
}

func (ctrl *DescargoController) PreviewFile(c *gin.Context) {
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

	ruta := c.Query("ruta")
	fecha := c.Query("fecha")
	boleto := c.Query("boleto")
	vuelo := c.Query("vuelo")

	tramoRegistrado := c.Query("info_tramo_registrado")
	fechaRegistrada := c.Query("info_fecha_registrada")
	boletoRegistrado := c.Query("info_boleto_registrado")
	paseRegistrado := c.Query("info_pase_registrado")

	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":                "Previsualización de Documento",
		"FilePath":             fullPath,
		"IsPDF":                isPDF,
		"InfoRuta":             ruta,
		"InfoFecha":            fecha,
		"InfoBoleto":           boleto,
		"InfoVuelo":            vuelo,
		"InfoTramoRegistrado":  tramoRegistrado,
		"InfoFechaRegistrada":  fechaRegistrada,
		"InfoBoletoRegistrado": boletoRegistrado,
		"InfoPaseRegistrado":   paseRegistrado,
	})
}
