package controllers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	agenciaService      *services.AgenciaService
	rutaService         *services.RutaService
	solicitudService    *services.SolicitudService
}

func NewPasajeController() *PasajeController {
	return &PasajeController{
		pasajeService:       services.NewPasajeService(),
		aerolineaService:    services.NewAerolineaService(),
		estadoPasajeService: services.NewEstadoPasajeService(),
		agenciaService:      services.NewAgenciaService(),
		rutaService:         services.NewRutaService(),
		solicitudService:    services.NewSolicitudService(),
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
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pasaje no encontrado"})
		} else {
			utils.SetErrorMessage(c, "Pasaje no encontrado")
			c.Redirect(http.StatusFound, "/solicitudes")
		}
		return
	}

	pasaje.EstadoPasajeCodigo = &status

	if status == "VALIDANDO_USO" {
		file, errFile := c.FormFile("archivo_pase_abordo")
		if errFile == nil {
			uploadDir := "uploads/pases_abordo"
			if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
				os.MkdirAll(uploadDir, 0755)
			}

			ext := filepath.Ext(file.Filename)
			filename := fmt.Sprintf("pase_%s_%d%s", pasaje.ID, time.Now().Unix(), ext)
			dst := filepath.Join(uploadDir, filename)

			if err := c.SaveUploadedFile(file, dst); err == nil {
				pasaje.ArchivoPaseAbordo = dst
			} else {
				log.Printf("Error saving boarding pass: %v", err)
			}
		}
	}

	if err := ctrl.pasajeService.Update(c.Request.Context(), pasaje); err != nil {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado"})
		} else {
			utils.SetErrorMessage(c, "Error al actualizar el estado")
			c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", pasaje.SolicitudID))
		}
		return
	}

	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		c.JSON(http.StatusOK, gin.H{"message": "Estado actualizado"})
	} else {
		msg := "Estado actualizado correctamente"
		switch status {
		case "VALIDANDO_USO":
			msg = "Documento enviado para validación"
		case "USADO":
			msg = "Uso del pasaje aprobado correctamente"
		case "USO_RECHAZADO":
			msg = "Uso del pasaje rechazado. El beneficiario deberá subirlo nuevamente."
		}
		utils.SetSuccessMessage(c, msg)
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", pasaje.SolicitudID))
	}
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

	filePase, errPase := c.FormFile("archivo_pase_abordo")
	if errPase == nil {
		uploadDir := "uploads/pases_abordo"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}
		ext := filepath.Ext(filePase.Filename)
		filename := fmt.Sprintf("pase_%s_%d_edit%s", pasaje.ID, time.Now().Unix(), ext)
		dst := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(filePase, dst); err == nil {
			pasaje.ArchivoPaseAbordo = dst
		}
	}

	ctrl.pasajeService.Update(c.Request.Context(), pasaje)

	utils.SetSuccessMessage(c, "Datos del pasaje actualizados")
	c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", pasaje.SolicitudID))
}

func (ctrl *PasajeController) Preview(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	tipo := c.Query("tipo")
	filePath := pasaje.Archivo
	title := "Vista Previa de Boleto"
	if tipo == "pase" {
		filePath = pasaje.ArchivoPaseAbordo
		title = "Vista Previa de Pase a Bordo"
	}

	if filePath == "" {
		c.String(http.StatusNotFound, "Archivo no disponible")
		return
	}

	c.HTML(http.StatusOK, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":    title,
		"FilePath": "/" + filePath,
		"IsPDF":    strings.HasSuffix(strings.ToLower(filePath), ".pdf"),
	})
}

func (ctrl *PasajeController) GetCreateModal(c *gin.Context) {
	solicitudID := c.Param("id")
	solicitud, err := ctrl.solicitudService.FindByID(c.Request.Context(), solicitudID)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	utils.Render(c, "solicitud/components/modal_crear_pasaje", gin.H{
		"Solicitud":  solicitud,
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
		"Rutas":      rutas,
	})
}

func (ctrl *PasajeController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	utils.Render(c, "solicitud/components/modal_editar_pasaje", gin.H{
		"Pasaje":     pasaje,
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
		"Rutas":      rutas,
	})
}

func (ctrl *PasajeController) GetReprogramarModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	utils.Render(c, "solicitud/components/modal_reprogramar_pasaje", gin.H{
		"Pasaje":     pasaje,
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
		"Rutas":      rutas,
	})
}

func (ctrl *PasajeController) GetDevolverModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	utils.Render(c, "solicitud/components/modal_devolver_pasaje", gin.H{
		"Pasaje": pasaje,
	})
}

func (ctrl *PasajeController) GetUsadoModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	utils.Render(c, "solicitud/components/modal_usado_pasaje", gin.H{
		"Pasaje": pasaje,
	})
}
