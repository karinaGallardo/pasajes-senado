package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"

	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type PasajeController struct {
	repo *repositories.PasajeRepository
}

func NewPasajeController() *PasajeController {
	return &PasajeController{
		repo: repositories.NewPasajeRepository(),
	}
}

func (ctrl *PasajeController) Store(c *gin.Context) {
	solicitudID, _ := strconv.Atoi(c.Param("id"))

	costo, _ := strconv.ParseFloat(c.PostForm("costo"), 64)

	fechaVuelo, _ := time.Parse("2006-01-02T15:04", c.PostForm("fecha_vuelo"))

	nuevoPasaje := models.Pasaje{
		SolicitudID:   uint(solicitudID),
		Aerolinea:     c.PostForm("aerolinea"),
		NumeroVuelo:   c.PostForm("numero_vuelo"),
		Ruta:          c.PostForm("ruta"),
		FechaVuelo:    fechaVuelo,
		CodigoReserva: c.PostForm("codigo_reserva"),
		NumeroBoleto:  c.PostForm("numero_boleto"),
		Costo:         costo,
		Estado:        "EMITIDO",
	}

	if err := ctrl.repo.Create(&nuevoPasaje); err != nil {
		log.Printf("Error creando pasaje: %v", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/%d?error=ErrorCrearPasaje", solicitudID))
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/%d?success=PasajeCreado", solicitudID))
}
