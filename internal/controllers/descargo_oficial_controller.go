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

	"github.com/gin-gonic/gin"
)

type DescargoOficialController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
	destinoService   *services.DestinoService
	reportService    *services.ReportService
	peopleService    *services.PeopleService
}

func NewDescargoOficialController() *DescargoOficialController {
	return &DescargoOficialController{
		descargoService:  services.NewDescargoService(),
		solicitudService: services.NewSolicitudService(),
		destinoService:   services.NewDestinoService(),
		reportService:    services.NewReportService(),
		peopleService:    services.NewPeopleService(),
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
	utils.Render(c, "descargo/oficial/create", gin.H{
		"Title":                "Nuevo Descargo (Oficial)",
		"Solicitud":            solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
		"HasGastosRep":         hasGastosRep,
		"Destinos":             destinos,
		"ZeroFloat":            float64(0),
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

	utils.Render(c, "descargo/oficial/show", gin.H{
		"Title":    "Detalle de Descargo (Oficial)",
		"Descargo": descargo,
	})
}

func (ctrl *DescargoOficialController) Edit(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	if descargo.Estado != "EN_REVISION" {
		c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
		return
	}

	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)
	for _, item := range descargo.DetallesItinerario {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
	}

	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	utils.Render(c, "descargo/oficial/edit", gin.H{
		"Title":       "Editar Descargo (Oficial)",
		"Descargo":    descargo,
		"ItemsByType": itemsByType,
		"Destinos":    destinos,
	})
}

func (ctrl *DescargoOficialController) Update(c *gin.Context) {
	id := c.Param("id")
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

func (ctrl *DescargoOficialController) Approve(c *gin.Context) {
	id := c.Param("id")
	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error aprobando descargo oficial: %v", err)
		c.Redirect(http.StatusFound, "/descargos")
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

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}

func (ctrl *DescargoOficialController) RevertApproval(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}

	if err := ctrl.descargoService.RevertApproval(c.Request.Context(), id, authUser.ID); err != nil {
		c.String(http.StatusInternalServerError, "Error revirtiendo aprobación: "+err.Error())
		return
	}

	descargo, _ := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if descargo != nil && descargo.SolicitudID != "" {
		if err := ctrl.solicitudService.RevertFinalize(c.Request.Context(), descargo.SolicitudID); err != nil {
			log.Printf("Warning: error reverting solicitud finalization: %v", err)
		}
	}

	c.Redirect(http.StatusFound, "/descargos/oficial/"+id)
}
