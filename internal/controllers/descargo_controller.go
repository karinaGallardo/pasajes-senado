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
	"time"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
)

type DescargoController struct {
	descargoService  *services.DescargoService
	solicitudService *services.SolicitudService
}

func NewDescargoController() *DescargoController {
	return &DescargoController{
		descargoService:  services.NewDescargoService(),
		solicitudService: services.NewSolicitudService(),
	}
}

func (ctrl *DescargoController) Index(c *gin.Context) {
	descargos, _ := ctrl.descargoService.FindAll(c.Request.Context())
	utils.Render(c, "descargo/index", gin.H{
		"Title":     "Bandeja de Descargos (FV-05)",
		"Descargos": descargos,
	})
}

func (ctrl *DescargoController) Create(c *gin.Context) {
	solicitudID := c.Query("solicitud_id")
	if solicitudID == "" {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	solicitud, err := ctrl.solicitudService.FindByID(c.Request.Context(), solicitudID)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	existe, _ := ctrl.descargoService.FindBySolicitudID(c.Request.Context(), solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/"+existe.ID)
		return
	}

	utils.Render(c, "descargo/create", gin.H{
		"Title":     "Nuevo Descargo",
		"Solicitud": solicitud,
	})
}

func (ctrl *DescargoController) Store(c *gin.Context) {
	var req dtos.CreateDescargoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes?error=DatosInvalidos")
		return
	}

	solicitudID := req.SolicitudID
	fechaPresentacion, err := time.Parse("2006-01-02", req.FechaPresentacion)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes?error=FechaInvalida")
		return
	}

	monto, err := strconv.ParseFloat(req.MontoDevolucion, 64)
	if err != nil {
		monto = 0
	}

	userContext := appcontext.CurrentUser(c)
	if userContext == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeSuffix, _ := gonanoid.Generate(alphabet, 6)
	codigo := fmt.Sprintf("D-%d-%s", time.Now().Year(), codeSuffix)

	nuevoDescargo := models.Descargo{
		SolicitudID:        solicitudID,
		UsuarioID:          userContext.ID,
		Codigo:             codigo,
		NumeroCite:         req.NumeroCite,
		FechaPresentacion:  fechaPresentacion,
		InformeActividades: req.InformeActividades,
		MontoDevolucion:    monto,
		Observaciones:      req.Observaciones,
		Estado:             "EN_REVISION",
	}
	nuevoDescargo.CreatedBy = &userContext.ID

	tipos := req.DocTipo
	numeros := req.DocNumero
	fechas := req.DocFecha
	detalles := req.DocDetalle

	var docs []models.DocumentoDescargo
	for i := range tipos {
		if i < len(numeros) && numeros[i] != "" {
			f, _ := time.Parse("2006-01-02", fechas[i])
			docs = append(docs, models.DocumentoDescargo{
				Tipo:    tipos[i],
				Numero:  numeros[i],
				Fecha:   f,
				Detalle: detalles[i],
			})
		}
	}
	nuevoDescargo.Documentos = docs

	if err := ctrl.descargoService.Create(c.Request.Context(), &nuevoDescargo); err != nil {
		log.Printf("Error creando descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	c.Redirect(http.StatusFound, "/descargos")
}

func (ctrl *DescargoController) Show(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.descargoService.FindByID(c.Request.Context(), id)
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

	descargo, err := ctrl.descargoService.FindByID(c.Request.Context(), id)
	if err != nil {
		log.Printf("Error aprobando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/"+id)
		return
	}

	descargo.Estado = "APROBADO"
	userContext := appcontext.CurrentUser(c)
	if userContext == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}
	descargo.UpdatedBy = &userContext.ID

	ctrl.descargoService.Update(c.Request.Context(), descargo)

	if descargo.SolicitudID != "" {
		ctrl.solicitudService.Finalize(c.Request.Context(), descargo.SolicitudID)
	}

	c.Redirect(http.StatusFound, "/descargos/"+id)
}
