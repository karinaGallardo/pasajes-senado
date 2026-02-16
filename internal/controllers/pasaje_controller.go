package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type PasajeController struct {
	agenciaService   *services.AgenciaService
	rutaService      *services.RutaService
	solicitudService *services.SolicitudService
	pasajeService    *services.PasajeService
	aerolineaService *services.AerolineaService
}

func NewPasajeController() *PasajeController {
	return &PasajeController{
		agenciaService:   services.NewAgenciaService(),
		rutaService:      services.NewRutaService(),
		solicitudService: services.NewSolicitudService(),
		pasajeService:    services.NewPasajeService(),
		aerolineaService: services.NewAerolineaService(),
	}
}

func (ctrl *PasajeController) Store(c *gin.Context) {
	solicitudID := c.Param("id")
	var req dtos.CreatePasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
		if solicitud != nil && solicitud.GetConceptoCodigo() == "OFICIAL" {
			c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/oficial/%s/detalle", solicitudID))
		} else {
			c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
		}
		return
	}

	var filePath string
	if file, err := c.FormFile("archivo"); err == nil {
		filePath, _ = utils.SaveUploadedFile(c, file, "uploads/pasajes", "pasaje_"+solicitudID+"_new_")
	}

	if _, err := ctrl.pasajeService.Create(c.Request.Context(), solicitudID, req, filePath); err != nil {
		utils.SetErrorMessage(c, "Error: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Pasaje registrado correctamente")
	}
	solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	if solicitud != nil && solicitud.GetConceptoCodigo() == "OFICIAL" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/oficial/%s/detalle", solicitudID))
	} else {
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID))
	}
}

func (ctrl *PasajeController) UpdateStatus(c *gin.Context) {
	var req dtos.UpdatePasajeStatusRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	var ticketPath, pasePath string
	switch req.Status {
	case "EMITIDO":
		if file, err := c.FormFile("archivo_ticket"); err == nil {
			ticketPath, _ = utils.SaveUploadedFile(c, file, "uploads/pasajes", "pasaje_"+req.ID+"_")
		}
	case "USADO":
		if file, err := c.FormFile("archivo_pase_abordo"); err == nil {
			pasePath, _ = utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_"+req.ID+"_")
		}
	}

	if err := ctrl.pasajeService.UpdateStatus(c.Request.Context(), req.ID, req.Status, ticketPath, pasePath); err != nil {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado"})
		} else {
			utils.SetErrorMessage(c, "Error al actualizar el estado")
			c.Redirect(http.StatusFound, "/solicitudes")
		}
		return
	}

	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		c.JSON(http.StatusOK, gin.H{"message": "Estado actualizado"})
	} else {
		msg := "Estado actualizado correctamente"
		switch req.Status {
		case "EMITIDO":
			msg = "Pasaje emitido correctamente"
		case "USADO":
			msg = "Pase a bordo registrado y validado"
		case "ANULADO":
			msg = "Pasaje anulado correctamente"
		}
		utils.SetSuccessMessage(c, msg)
		c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
	}
}

func (ctrl *PasajeController) Devolver(c *gin.Context) {
	var req dtos.DevolverPasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	if err := ctrl.pasajeService.DevolverPasaje(c.Request.Context(), req); err != nil {
		utils.SetErrorMessage(c, "Error: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	utils.SetSuccessMessage(c, "Pasaje marcado como DEVUELTO correctamente")
	c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
}

func (ctrl *PasajeController) Update(c *gin.Context) {
	var req dtos.UpdatePasajeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	var filePath, pasePath string
	if file, err := c.FormFile("archivo"); err == nil {
		filePath, _ = utils.SaveUploadedFile(c, file, "uploads/pasajes", "pasaje_"+req.ID+"_edit_")
	}
	if file, err := c.FormFile("archivo_pase_abordo"); err == nil {
		pasePath, _ = utils.SaveUploadedFile(c, file, "uploads/pases_abordo", "pase_"+req.ID+"_edit_")
	}

	if err := ctrl.pasajeService.UpdateFromRequest(c.Request.Context(), req, filePath, pasePath); err != nil {
		utils.SetErrorMessage(c, "Error: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Datos del pasaje actualizados")
	}

	c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
}

func (ctrl *PasajeController) Preview(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
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
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	utils.Render(c, "solicitud/components/modal_crear_pasaje", gin.H{
		"Solicitud":       solicitud,
		"Aerolineas":      aerolineas,
		"Agencias":        agencias,
		"Rutas":           rutas,
		"SolicitudItemID": c.Query("item_id"),
	})
}

func (ctrl *PasajeController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
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

func (ctrl *PasajeController) GetDevolverModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
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
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	utils.Render(c, "solicitud/components/modal_usado_pasaje", gin.H{
		"Pasaje": pasaje,
	})
}
