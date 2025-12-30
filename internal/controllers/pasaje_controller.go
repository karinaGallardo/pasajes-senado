package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"strconv"

	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type PasajeController struct {
	pasajeService    *services.PasajeService
	aerolineaService *services.AerolineaService
}

func NewPasajeController() *PasajeController {
	return &PasajeController{
		pasajeService:    services.NewPasajeService(),
		aerolineaService: services.NewAerolineaService(),
	}
}

func (ctrl *PasajeController) Store(c *gin.Context) {
	solicitudID := c.Param("id")

	var req dtos.CreatePasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/%s?error=DatosInvalidos", solicitudID))
		return
	}

	costo, _ := strconv.ParseFloat(req.Costo, 64)
	fechaVuelo, _ := time.Parse("2006-01-02T15:04", req.FechaVuelo)

	aerolineaNombre := req.Aerolinea
	var aerolineaID *string
	if aerolineaNombre != "" {
		all, _ := ctrl.aerolineaService.GetAllActive()
		for _, a := range all {
			if a.Nombre == aerolineaNombre {
				id := a.ID
				aerolineaID = &id
				break
			}
		}
	}

	nuevoPasaje := models.Pasaje{
		SolicitudID:   solicitudID,
		AerolineaID:   aerolineaID,
		NumeroVuelo:   req.NumeroVuelo,
		Ruta:          req.Ruta,
		FechaVuelo:    fechaVuelo,
		CodigoReserva: req.CodigoReserva,
		NumeroBoleto:  req.NumeroBoleto,
		Costo:         costo,
		Estado:        "EMITIDO",
	}

	if err := ctrl.pasajeService.Create(&nuevoPasaje); err != nil {
		log.Printf("Error creando pasaje: %v", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/%s?error=ErrorCrearPasaje", solicitudID))
		return
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/%s?success=PasajeCreado", solicitudID))
}
