package controllers

import (
	"fmt"
	"log"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

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
		"Title":     "Bandeja de Descargos (FV-05)",
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
		} else {
			// Fallback to item route if no pasaje yet (unlikely for descargo)
			pasajesOriginales[tipo] = append(pasajesOriginales[tipo], ConnectionView{
				Ruta:  item.OrigenIATA + " - " + item.DestinoIATA,
				Fecha: "",
			})
			if item.Fecha != nil {
				pasajesOriginales[tipo][0].Fecha = item.Fecha.Format("2006-01-02")
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

	utils.Render(c, "descargo/create", gin.H{
		"Title":                "Nuevo Descargo",
		"Solicitud":            solicitud,
		"PasajesOriginales":    pasajesOriginales,
		"PasajesReprogramados": pasajesReprogramados,
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

	if _, err := ctrl.descargoService.Create(c.Request.Context(), req, authUser.ID, archivoPaths); err != nil {
		log.Printf("Error creando descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=ErrorCreacion")
		return
	}

	c.Redirect(http.StatusFound, "/descargos")
}

func (ctrl *DescargoController) Show(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.GetByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error buscando descargo %s: %v", id, err)
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	utils.Render(c, "descargo/show", gin.H{
		"Title":    "Detalle de Descargo",
		"Descargo": descargo,
	})
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

	// Generar PV5 con Boarding Passes unidos
	pdfReader, err := ctrl.reportService.GeneratePV05Complete(c.Request.Context(), descargo, personaView)
	if err != nil {
		log.Printf("Error generating complete PDF: %v", err)
		c.String(http.StatusInternalServerError, "Error generando PDF")
		return
	}

	filename := "PV5_Descargo_" + descargo.Codigo + ".pdf"
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

	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    "Previsualización Formulario PV-05",
		"FilePath": fmt.Sprintf("/descargos/%s/imprimir-pv5", descargo.ID),
		"IsPDF":    true,
	})
}
