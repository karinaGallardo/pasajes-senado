package controllers

import (
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	repo       *repositories.SolicitudRepository
	ciudadRepo *repositories.CiudadRepository
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		repo:       repositories.NewSolicitudRepository(),
		ciudadRepo: repositories.NewCiudadRepository(),
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.repo.FindAll()
	c.HTML(http.StatusOK, "solicitud/index.html", gin.H{
		"Title":       "Bandeja de Solicitudes",
		"Solicitudes": solicitudes,
		"User":        c.MustGet("User"),
	})
}

func (ctrl *SolicitudController) Create(c *gin.Context) {
	destinos, _ := ctrl.ciudadRepo.FindAll()
	c.HTML(http.StatusOK, "solicitud/create.html", gin.H{
		"Title":    "Nueva Solicitud de Pasaje",
		"User":     c.MustGet("User"),
		"Destinos": destinos,
	})
}

func (ctrl *SolicitudController) Store(c *gin.Context) {
	layout := "2006-01-02T15:04"
	fechaSalida, _ := time.Parse(layout, c.PostForm("fecha_salida"))

	var fechaRetorno time.Time
	if c.PostForm("fecha_retorno") != "" {
		fechaRetorno, _ = time.Parse(layout, c.PostForm("fecha_retorno"))
	}

	userContext, exists := c.Get("User")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	usuario := userContext.(models.Usuario)

	nuevaSolicitud := models.Solicitud{
		UsuarioID:    usuario.ID,
		TipoViaje:    c.PostForm("tipo_viaje"),
		OrigenCode:   c.PostForm("origen"),
		DestinoCode:  c.PostForm("destino"),
		FechaSalida:  fechaSalida,
		FechaRetorno: fechaRetorno,
		Motivo:       c.PostForm("motivo"),
		Estado:       "SOLICITADO",
	}

	if err := ctrl.repo.Create(&nuevaSolicitud); err != nil {
		c.HTML(http.StatusInternalServerError, "solicitud/create.html", gin.H{
			"Error": "No se pudo crear la solicitud: " + err.Error(),
			"User":  c.MustGet("User"),
		})
		return
	}

	c.Redirect(http.StatusFound, "/solicitudes")
}

func (ctrl *SolicitudController) Show(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	solicitud, err := ctrl.repo.FindByID(uint(id))

	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	c.HTML(http.StatusOK, "solicitud/show.html", gin.H{
		"Title":     "Detalle Solicitud #" + strconv.Itoa(id),
		"Solicitud": solicitud,
		"User":      c.MustGet("User"),
	})
}
