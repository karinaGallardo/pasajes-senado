package controllers

import (
	"fmt"
	"net/http"
	"strings"

	"sistema-pasajes/internal/appcontext"
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

func NewPasajeController(
	agenciaService *services.AgenciaService,
	rutaService *services.RutaService,
	solicitudService *services.SolicitudService,
	pasajeService *services.PasajeService,
	aerolineaService *services.AerolineaService,
) *PasajeController {
	return &PasajeController{
		agenciaService:   agenciaService,
		rutaService:      rutaService,
		solicitudService: solicitudService,
		pasajeService:    pasajeService,
		aerolineaService: aerolineaService,
	}
}

func (ctrl *PasajeController) Store(c *gin.Context) {
	solicitudID := c.Param("id")
	var req dtos.CreatePasajeRequest

	isHTMX := c.GetHeader("HX-Request") == "true"

	if err := c.ShouldBind(&req); err != nil {
		if isHTMX {
			ctrl.renderCreateModalWithError(c, solicitudID, req, "Datos inválidos: verifique los campos obligatorios")
			return
		}
		utils.SetErrorMessage(c, "Datos inválidos")
		solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
		if solicitud != nil && solicitud.IsOficial() {
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
		if isHTMX {
			ctrl.renderCreateModalWithError(c, solicitudID, req, "Error al guardar: "+err.Error())
			return
		}
		utils.SetErrorMessage(c, "Error: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Pasaje registrado correctamente")
	}

	solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	targetURL := ""
	if solicitud != nil && solicitud.IsOficial() {
		targetURL = fmt.Sprintf("/solicitudes/oficial/%s/detalle", solicitudID)
	} else {
		targetURL = fmt.Sprintf("/solicitudes/derecho/%s/detalle", solicitudID)
	}

	if isHTMX {
		c.Header("HX-Location", targetURL)
		c.Status(204)
		return
	}

	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *PasajeController) renderCreateModalWithError(c *gin.Context, solicitudID string, req dtos.CreatePasajeRequest, errMsg string) {
	solicitud, _ := ctrl.solicitudService.GetByID(c.Request.Context(), solicitudID)
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	authUser := appcontext.AuthUser(c)
	solicitud.HydratePermissions(authUser)

	utils.Render(c, "solicitud/components/modal_registrar_pasaje", gin.H{
		"Solicitud":       solicitud,
		"Aerolineas":      aerolineas,
		"Agencias":        agencias,
		"Rutas":           rutas,
		"SolicitudItemID": req.SolicitudItemID,
		"Form":            req,
		"ErrorMessage":    errMsg,
		"Fares":           ctrl.rutaService.GetFaresMap(rutas),
	})
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
		if c.GetHeader("HX-Request") == "true" {
			// Find which modal to re-render based on status
			if req.Status == "EMITIDO" {
				// Re-rendering emitir logic might need a separate helper or manual render
				// For now, simple JSON error is fine if it's handled by alert, but re-render is better.
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado: " + err.Error()})
			return
		}
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar estado"})
		} else {
			utils.SetErrorMessage(c, "Error al actualizar el estado")
			c.Redirect(http.StatusFound, "/solicitudes")
		}
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Refresh", "true")
		c.Status(204)
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
	isHTMX := c.GetHeader("HX-Request") == "true"

	if err := c.ShouldBind(&req); err != nil {
		if isHTMX {
			ctrl.renderEditModalWithError(c, req.ID, req, "Datos inválidos")
			return
		}
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
		if isHTMX {
			ctrl.renderEditModalWithError(c, req.ID, req, "Error: "+err.Error())
			return
		}
		utils.SetErrorMessage(c, "Error: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Datos del pasaje actualizados")
	}

	if isHTMX {
		c.Header("HX-Refresh", "true")
		c.Status(204)
		return
	}

	c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
}

func (ctrl *PasajeController) renderEditModalWithError(c *gin.Context, pasajeID string, req dtos.UpdatePasajeRequest, errMsg string) {
	pasaje, _ := ctrl.pasajeService.GetByID(c.Request.Context(), pasajeID)
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	agencias, _ := ctrl.agenciaService.GetAllActive(c.Request.Context())
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())

	authUser := appcontext.AuthUser(c)
	pasaje.HydratePermissions(authUser)

	utils.Render(c, "solicitud/components/modal_editar_pasaje", gin.H{
		"Pasaje":       pasaje,
		"Aerolineas":   aerolineas,
		"Agencias":     agencias,
		"Rutas":        rutas,
		"Form":         req,
		"ErrorMessage": errMsg,
		"Fares":        ctrl.rutaService.GetFaresMap(rutas),
	})
}

func (ctrl *PasajeController) Preview(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	tipo := c.Query("tipo")
	var files []gin.H

	switch tipo {
	case "billete":
		if pasaje.Archivo != "" {
			files = append(files, gin.H{
				"Title": "Billete de Pasaje",
				"Path":  "/" + pasaje.Archivo,
				"IsPDF": strings.HasSuffix(strings.ToLower(pasaje.Archivo), ".pdf"),
			})
		}
		if pasaje.ServicioArchivo != "" {
			files = append(files, gin.H{
				"Title": "Factura de Servicio de Emisión",
				"Path":  "/" + pasaje.ServicioArchivo,
				"IsPDF": strings.HasSuffix(strings.ToLower(pasaje.ServicioArchivo), ".pdf"),
			})
		}
	case "pase":
		if pasaje.ArchivoPaseAbordo != "" {
			files = append(files, gin.H{
				"Title": "Pase a Bordo",
				"Path":  "/" + pasaje.ArchivoPaseAbordo,
				"IsPDF": strings.HasSuffix(strings.ToLower(pasaje.ArchivoPaseAbordo), ".pdf"),
			})
		}
	}

	if len(files) == 0 {
		c.String(http.StatusNotFound, "Archivo no disponible")
		return
	}

	utils.Render(c, "solicitud/components/modal_preview_archivo", gin.H{
		"Title":  "Vista Previa de Documentos",
		"Files":  files,
		"Pasaje": pasaje,
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

	itemID := c.Query("item_id")
	authUser := appcontext.AuthUser(c)
	solicitud.HydratePermissions(authUser)
	selectedItem := solicitud.GetItemByID(itemID)
	if selectedItem != nil {
		selectedItem.HydratePermissions(authUser)
	}

	utils.Render(c, "solicitud/components/modal_registrar_pasaje", gin.H{
		"Solicitud":    solicitud,
		"Aerolineas":   aerolineas,
		"Agencias":     agencias,
		"Rutas":        rutas,
		"SelectedItem": selectedItem,
		"Fares":        ctrl.rutaService.GetFaresMap(rutas),
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

	authUser := appcontext.AuthUser(c)
	pasaje.HydratePermissions(authUser)

	utils.Render(c, "solicitud/components/modal_editar_pasaje", gin.H{
		"Pasaje":     pasaje,
		"Aerolineas": aerolineas,
		"Agencias":   agencias,
		"Rutas":      rutas,
		"Fares":      ctrl.rutaService.GetFaresMap(rutas),
	})
}

func (ctrl *PasajeController) GetDevolverModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	authUser := appcontext.AuthUser(c)
	pasaje.HydratePermissions(authUser)

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

	authUser := appcontext.AuthUser(c)
	pasaje.HydratePermissions(authUser)

	utils.Render(c, "solicitud/components/modal_usado_pasaje", gin.H{
		"Pasaje": pasaje,
	})
}

func (ctrl *PasajeController) Delete(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	if err := ctrl.pasajeService.Delete(c.Request.Context(), id, authUser.ID); err != nil {
		if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		} else {
			utils.SetErrorMessage(c, "Error: "+err.Error())
			c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
		}
		return
	}

	if c.GetHeader("X-Requested-With") == "XMLHttpRequest" {
		c.Status(http.StatusOK)
	} else {
		utils.SetSuccessMessage(c, "Pasaje eliminado correctamente")
		c.Redirect(http.StatusFound, c.Request.Header.Get("Referer"))
	}
}

func (ctrl *PasajeController) GetServicioModal(c *gin.Context) {
	id := c.Param("id")
	pasaje, err := ctrl.pasajeService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Pasaje no encontrado")
		return
	}

	utils.Render(c, "solicitud/components/modal_servicio_pasaje", gin.H{
		"Pasaje": pasaje,
	})
}

func (ctrl *PasajeController) UpdateServicio(c *gin.Context) {
	var req dtos.UpdateServicioEmisionRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	var filePath string
	if file, err := c.FormFile("servicio_archivo"); err == nil {
		filePath, _ = utils.SaveUploadedFile(c, file, "uploads/servicios", "servicio_"+req.ID+"_")
	}

	if err := ctrl.pasajeService.UpdateServicioEmision(c.Request.Context(), req, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}
