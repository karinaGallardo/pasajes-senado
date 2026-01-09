package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type PasajeController struct {
	pasajeService       *services.PasajeService
	aerolineaService    *services.AerolineaService
	estadoPasajeService *services.EstadoPasajeService
}

func NewPasajeController() *PasajeController {
	return &PasajeController{
		pasajeService:       services.NewPasajeService(),
		aerolineaService:    services.NewAerolineaService(),
		estadoPasajeService: services.NewEstadoPasajeService(),
	}
}

func (ctrl *PasajeController) Store(c *gin.Context) {
	solicitudID := c.Param("id")

	var req dtos.CreatePasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		utils.SetErrorMessage(c, "Datos inválidos en el formulario")
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
		return
	}

	var filePath string
	file, errFile := c.FormFile("archivo")
	if errFile == nil {
		uploadDir := "uploads/pasajes"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}

		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("pasaje_%s_%d%s", solicitudID, time.Now().Unix(), ext)
		dst := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, dst); err == nil {
			filePath = dst
		} else {
			log.Printf("Error saving file: %v", err)
			utils.SetErrorMessage(c, "Error al guardar el archivo adjunto")
			c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
			return
		}
	}

	costo, _ := strconv.ParseFloat(req.Costo, 64)
	fechaVuelo, _ := time.Parse("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if id := c.PostForm("aerolinea_id"); id != "" {
		aerolineaID = &id
	}

	nuevoPasaje := models.Pasaje{
		SolicitudID:   solicitudID,
		AerolineaID:   aerolineaID,
		AgenciaID:     &req.AgenciaID,
		NumeroVuelo:   req.NumeroVuelo,
		Ruta:          req.Ruta,
		FechaVuelo:    fechaVuelo,
		CodigoReserva: req.CodigoReserva,
		NumeroBoleto:  req.NumeroBoleto,
		NumeroFactura: req.NumeroFactura,
		Glosa:         req.Glosa,
		Costo:         costo,

		Archivo: filePath,
	}

	if err := ctrl.pasajeService.Create(c.Request.Context(), &nuevoPasaje); err != nil {
		log.Printf("Error creando pasaje: %v", err)
		utils.SetErrorMessage(c, "Error al crear el pasaje: "+err.Error())
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
		return
	}

	utils.SetSuccessMessage(c, "Pasaje registrado correctamente")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
}
func (ctrl *PasajeController) UpdateStatus(c *gin.Context) {
	id := c.PostForm("id")
	status := c.PostForm("status")

	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pasaje no encontrado"})
		return
	}

	pasaje.EstadoPasajeCodigo = &status

	if err := ctrl.pasajeService.Update(c.Request.Context(), pasaje); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Estado actualizado"})
}

func (ctrl *PasajeController) Devolver(c *gin.Context) {
	id := c.PostForm("pasaje_id")
	glosa := c.PostForm("glosa")
	penalidadStr := c.PostForm("costo_penalidad")

	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	costoPenalidad, _ := strconv.ParseFloat(penalidadStr, 64)

	newState := "DEVUELTO"

	pasaje.EstadoPasajeCodigo = &newState

	if pasaje.Glosa != "" {
		pasaje.Glosa += " | Devolución: " + glosa
	} else {
		pasaje.Glosa = "Devolución: " + glosa
	}
	pasaje.CostoPenalidad = costoPenalidad

	if err := ctrl.pasajeService.Update(c.Request.Context(), pasaje); err != nil {
		log.Printf("Error devolviendo pasaje: %v", err)
	}

	utils.SetSuccessMessage(c, "Pasaje marcado como DEVUELTO correctamente")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", pasaje.SolicitudID))
}

func (ctrl *PasajeController) Reprogramar(c *gin.Context) {
	var req dtos.ReprogramarPasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Error binding request: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes?error=DatosInvalidos")
		return
	}

	var filePath string
	file, errFile := c.FormFile("archivo")
	if errFile == nil {
		uploadDir := "uploads/pasajes"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}

		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("pasaje_%s_%d_reprog%s", req.PasajeAnteriorID, time.Now().Unix(), ext)
		dst := filepath.Join(uploadDir, filename)

		if err := c.SaveUploadedFile(file, dst); err == nil {
			filePath = dst
		}
	}

	reqCosto, _ := strconv.ParseFloat(req.Costo, 64)
	reqPenalidad, _ := strconv.ParseFloat(req.CostoPenalidad, 64)
	reqFecha, _ := time.Parse("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	newPasaje := models.Pasaje{
		AerolineaID:    aerolineaID,
		AgenciaID:      &req.AgenciaID,
		NumeroVuelo:    req.NumeroVuelo,
		Ruta:           req.Ruta,
		FechaVuelo:     reqFecha,
		NumeroBoleto:   req.NumeroBoleto,
		Costo:          reqCosto,
		CostoPenalidad: reqPenalidad,
		Archivo:        filePath,
		Glosa:          req.Glosa,
		NumeroFactura:  req.NumeroFactura,
		CodigoReserva:  req.CodigoReserva,
	}

	if err := ctrl.pasajeService.Reprogramar(c.Request.Context(), req.PasajeAnteriorID, &newPasaje); err != nil {
		log.Printf("Error reprogramando pasaje: %v", err)
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	utils.SetSuccessMessage(c, "Pasaje reprogramado exitosamente")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", newPasaje.SolicitudID))
}

func (ctrl *PasajeController) Update(c *gin.Context) {
	id := c.PostForm("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	pasaje.Ruta = c.PostForm("ruta")
	pasaje.NumeroVuelo = c.PostForm("numero_vuelo")
	pasaje.NumeroBoleto = c.PostForm("numero_boleto")
	pasaje.NumeroFactura = c.PostForm("numero_factura")
	pasaje.CodigoReserva = c.PostForm("codigo_reserva")
	pasaje.Glosa = c.PostForm("glosa")

	if aerolineaID := c.PostForm("aerolinea_id"); aerolineaID != "" {
		pasaje.AerolineaID = &aerolineaID
	}

	if val, err := strconv.ParseFloat(c.PostForm("costo"), 64); err == nil {
		pasaje.Costo = val
	}

	if t, err := time.Parse("2006-01-02T15:04", c.PostForm("fecha_vuelo")); err == nil {
		pasaje.FechaVuelo = t
	}

	file, errFile := c.FormFile("archivo")
	if errFile == nil {
		uploadDir := "uploads/pasajes"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}
		ext := filepath.Ext(file.Filename)
		filename := fmt.Sprintf("pasaje_%s_%d_edit%s", pasaje.SolicitudID, time.Now().Unix(), ext)
		dst := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(file, dst); err == nil {
			pasaje.Archivo = dst
		}
	}

	ctrl.pasajeService.Update(c.Request.Context(), pasaje)

	utils.SetSuccessMessage(c, "Datos del pasaje actualizados")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", pasaje.SolicitudID))
}
