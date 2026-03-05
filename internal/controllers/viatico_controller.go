package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type ViaticoController struct {
	viaticoService   *services.ViaticoService
	solService       *services.SolicitudService
	categoriaService *services.CategoriaViaticoService
	reportService    *services.ReportService
}

func NewViaticoController(
	viaticoService *services.ViaticoService,
	solService *services.SolicitudService,
	categoriaService *services.CategoriaViaticoService,
	reportService *services.ReportService,
) *ViaticoController {
	return &ViaticoController{
		viaticoService:   viaticoService,
		solService:       solService,
		categoriaService: categoriaService,
		reportService:    reportService,
	}
}

func (ctrl *ViaticoController) Index(c *gin.Context) {
	viaticos, err := ctrl.viaticoService.GetByContext(c.Request.Context())
	if err != nil {
		c.String(500, "Error listando viáticos: "+err.Error())
		return
	}

	utils.Render(c, "viatico/index", gin.H{
		"Title":    "Gestión de Viáticos",
		"Viaticos": viaticos,
	})
}

func (ctrl *ViaticoController) Create(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	dias := 0.0
	fIda := solicitud.GetFechaIda()
	fVuelta := solicitud.GetFechaVuelta()

	if fVuelta != nil && fIda != nil {
		diff := fVuelta.Sub(*fIda)
		dias = diff.Hours() / 24
		if dias < 1 {
			dias = 1
		}
	} else {
		dias = 1
	}

	categorias, _ := ctrl.viaticoService.GetCategorias(c.Request.Context())
	zonas, _ := ctrl.viaticoService.GetZonas(c.Request.Context())

	utils.Render(c, "viatico/modal_create", gin.H{
		"Title":       "Asignación de Viáticos",
		"Solicitud":   solicitud,
		"DefaultDias": dias,
		"Categorias":  categorias,
		"Zonas":       zonas,
	})
}

func (ctrl *ViaticoController) Calculate(c *gin.Context) {
}

func (ctrl *ViaticoController) Store(c *gin.Context) {
	solicitudID := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var req dtos.CreateViaticoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	if _, err := ctrl.viaticoService.RegistrarViatico(c.Request.Context(), solicitudID, req, authUser.ID); err != nil {
		c.String(http.StatusInternalServerError, "Error asignando viático: "+err.Error())
		return
	}

	sol, _ := ctrl.solService.GetByID(c.Request.Context(), solicitudID)
	path := "derecho"
	if sol != nil && sol.GetConceptoCodigo() == "OFICIAL" {
		path = "oficial"
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+path+"/"+solicitudID+"/detalle")
}

func (ctrl *ViaticoController) Print(c *gin.Context) {
	id := c.Param("id")
	viatico, err := ctrl.viaticoService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving viatico: "+err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		if c.Query("download") == "1" {
			// HTMX descarga: redirigir al mismo endpoint sin HX-Request para que el navegador reciba el PDF y lo descargue
			c.Header("HX-Redirect", fmt.Sprintf("/viaticos/%s/print?download=1", id))
			c.Status(http.StatusOK)
			return
		}
		utils.Render(c, "viatico/modal_print", gin.H{
			"Viatico": viatico,
		})
		return
	}

	pdf := ctrl.reportService.GenerateViaticoV1(c.Request.Context(), viatico)

	c.Header("Content-Type", "application/pdf")
	filename := fmt.Sprintf("VIATICO-%s.pdf", viatico.Codigo)
	if c.Query("download") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", filename))
	}
	pdf.Output(c.Writer)
}
