package controllers

import (
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
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
	id := c.Param("id")
	solicitud, err := ctrl.repo.FindByID(id)

	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	st := solicitud.Estado
	step1 := true
	step2 := st == "APROBADO" || st == "FINALIZADO"
	step3 := st == "FINALIZADO"

	c.HTML(http.StatusOK, "solicitud/show.html", gin.H{
		"Title":     "Detalle Solicitud #" + id,
		"Solicitud": solicitud,
		"User":      c.MustGet("User"),
		"Step1":     step1,
		"Step2":     step2,
		"Step3":     step3,
	})
}
