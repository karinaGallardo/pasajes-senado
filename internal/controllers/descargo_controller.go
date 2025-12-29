package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type DescargoController struct {
	repo          *repositories.DescargoRepository
	solicitudRepo *repositories.SolicitudRepository
}

func NewDescargoController() *DescargoController {
	return &DescargoController{
		repo:          repositories.NewDescargoRepository(),
		solicitudRepo: repositories.NewSolicitudRepository(),
	}
}

func (ctrl *DescargoController) Index(c *gin.Context) {
	descargos, _ := ctrl.repo.FindAll()
	c.HTML(http.StatusOK, "descargo/index.html", gin.H{
		"Title":     "Bandeja de Descargos (FV-05)",
		"Descargos": descargos,
		"User":      c.MustGet("User"),
	})
}

func (ctrl *DescargoController) Create(c *gin.Context) {
	solicitudID := c.Query("solicitud_id")
	if solicitudID == "" {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	solicitud, err := ctrl.solicitudRepo.FindByID(solicitudID)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	existe, _ := ctrl.repo.FindBySolicitudID(solicitudID)
	if existe != nil && existe.ID != "" {
		c.Redirect(http.StatusFound, "/descargos/"+existe.ID)
		return
	}

	c.HTML(http.StatusOK, "descargo/create.html", gin.H{
		"Title":     "Nuevo Descargo",
		"Solicitud": solicitud,
		"User":      c.MustGet("User"),
	})
}

func (ctrl *DescargoController) Store(c *gin.Context) {
	solicitudID := c.PostForm("solicitud_id")

	fechaPresentacion, _ := time.Parse("2006-01-02", c.PostForm("fecha_presentacion"))

	monto, _ := strconv.ParseFloat(c.PostForm("monto_devolucion"), 64)

	userContext := c.MustGet("User").(models.Usuario)

	nuevoDescargo := models.Descargo{
		SolicitudID:        solicitudID,
		UsuarioID:          userContext.ID,
		FechaPresentacion:  fechaPresentacion,
		InformeActividades: c.PostForm("informe_actividades"),
		MontoDevolucion:    monto,
		Observaciones:      c.PostForm("observaciones"),
		Estado:             "EN_REVISION",
	}

	if err := ctrl.repo.Create(&nuevoDescargo); err != nil {
		log.Printf("Error creando descargo: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	c.Redirect(http.StatusFound, "/descargos")
}

func (ctrl *DescargoController) Show(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.repo.FindByID(id)
	if err != nil {
		log.Printf("Error buscando descargo %s: %v", id, err)
		c.Redirect(http.StatusFound, "/descargos")
		return
	}

	c.HTML(http.StatusOK, "descargo/show.html", gin.H{
		"Title":    "Detalle de Descargo",
		"Descargo": descargo,
		"User":     c.MustGet("User"),
	})
	c.HTML(http.StatusOK, "descargo/show.html", gin.H{
		"Title":    "Detalle de Descargo",
		"Descargo": descargo,
		"User":     c.MustGet("User"),
	})
}

func (ctrl *DescargoController) Approve(c *gin.Context) {
	id := c.Param("id")

	descargo, err := ctrl.repo.FindByID(id)
	if err != nil {
		log.Printf("Error aprobando descargo: %v", err)
		c.Redirect(http.StatusFound, "/descargos/"+id)
		return
	}

	descargo.Estado = "APROBADO"
	ctrl.repo.Update(descargo)

	if descargo.SolicitudID != "" {
		solicitud, err := ctrl.solicitudRepo.FindByID(descargo.SolicitudID)
		if err == nil {
			solicitud.Estado = "FINALIZADO"
			ctrl.solicitudRepo.Update(solicitud)
		}
	}

	c.Redirect(http.StatusFound, "/descargos/"+id)
}
