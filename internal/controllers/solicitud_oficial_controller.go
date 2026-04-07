package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"html/template"

	"github.com/gin-gonic/gin"
)

type SolicitudOficialController struct {
	solicitudService        *services.SolicitudService
	solicitudOficialService *services.SolicitudOficialService
	destinoService          *services.DestinoService
	tipoSolicitudService    *services.TipoSolicitudService
	ambitoService           *services.AmbitoService
	userService             *services.UsuarioService
	tipoItinerarioService   *services.TipoItinerarioService
	aerolineaService        *services.AerolineaService
	reportService           *services.ReportService
	peopleService           *services.PeopleService
	descargoService         *services.DescargoService
}

func NewSolicitudOficialController(
	solicitudService *services.SolicitudService,
	solicitudOficialService *services.SolicitudOficialService,
	destinoService *services.DestinoService,
	tipoSolicitudService *services.TipoSolicitudService,
	ambitoService *services.AmbitoService,
	userService *services.UsuarioService,
	tipoItinerarioService *services.TipoItinerarioService,
	aerolineaService *services.AerolineaService,
	reportService *services.ReportService,
	peopleService *services.PeopleService,
	descargoService *services.DescargoService,
) *SolicitudOficialController {
	return &SolicitudOficialController{
		solicitudService:        solicitudService,
		solicitudOficialService: solicitudOficialService,
		destinoService:          destinoService,
		tipoSolicitudService:    tipoSolicitudService,
		ambitoService:           ambitoService,
		userService:             userService,
		tipoItinerarioService:   tipoItinerarioService,
		aerolineaService:        aerolineaService,
		reportService:           reportService,
		peopleService:           peopleService,
		descargoService:         descargoService,
	}
}

func (ctrl *SolicitudOficialController) GetCreateModal(c *gin.Context) {
	authUser := appcontext.AuthUser(c)

	// Fetch necessary data for the form
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	ambitos, _ := ctrl.ambitoService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	tipos, _ := ctrl.tipoSolicitudService.GetByConcepto(c.Request.Context(), "OFICIAL")

	// Default target user is the logged in user
	targetUser := authUser

	// check if "target_user_id" is passed
	targetID := c.Query("target_user_id")
	if targetID != "" {
		u, err := ctrl.userService.GetByID(c.Request.Context(), targetID)
		if err == nil {
			targetUser = u
		}
	}

	dateIda := c.Query("fecha")

	render := "solicitud/oficial/modal_create"

	utils.Render(c, render, gin.H{
		"TargetUser":  targetUser,
		"Aerolineas":  aerolineas,
		"Ambitos":     ambitos,
		"Destinos":    destinos,
		"Tipos":       tipos,
		"DefaultDate": dateIda,
	})
}

func (ctrl *SolicitudOficialController) Store(c *gin.Context) {
	authUser := appcontext.AuthUser(c)

	var req dtos.CreateSolicitudOficialRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if js := c.PostForm("tramos_ida_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.TramosIda = tramos
		}
	}
	if js := c.PostForm("tramos_vuelta_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.TramosVuelta = tramos
		}
	}

	sol, err := ctrl.solicitudOficialService.CreateOficial(c.Request.Context(), req, authUser)
	if err != nil {
		utils.SetErrorMessage(c, "Error al crear la solicitud: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial")
		return
	}

	utils.SetSuccessMessage(c, "Solicitud de Comisión Oficial creada correctamente")

	estado := "SOLICITADO"
	if sol.EstadoSolicitudCodigo != nil {
		estado = *sol.EstadoSolicitudCodigo
	}
	c.Redirect(http.StatusFound, "/solicitudes/oficial?status="+estado)
}

func (ctrl *SolicitudOficialController) Show(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.HTML(http.StatusNotFound, "errors/404", gin.H{"Title": "Solicitud no encontrada"})
		return
	}

	if !solicitud.CanView(authUser) {
		c.HTML(http.StatusForbidden, "errors/403", gin.H{"Title": "No autorizado"})
		return
	}

	solicitud.HydratePermissions(authUser)
	steps, showNextSteps := solicitud.GetStepperData()
	statusCard := solicitud.GetStatusCardData()

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	// Descargo PV-05/06: si ya existe, pasamos ID y Estado para enlaces directos
	var descargoID string
	var descargoEstado string
	if descargo, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), id); descargo != nil && descargo.ID != "" {
		descargoID = descargo.ID
		descargoEstado = string(descargo.Estado)
	}

	utils.Render(c, "solicitud/oficial/show", gin.H{
		"Title":          "Solicitud de Comisión Oficial " + solicitud.Codigo,
		"Solicitud":      solicitud,
		"Steps":          steps,
		"ShowNextSteps":  showNextSteps,
		"StatusCard":     statusCard,
		"Aerolineas":     aerolineas,
		"DescargoID":     descargoID,
		"DescargoEstado": descargoEstado,
	})
}

func (ctrl *SolicitudOficialController) Approve(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.Approve(c.Request.Context(), id); err != nil {
		utils.SetErrorMessage(c, "Error al aprobar la solicitud: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Solicitud APROBADA correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) RevertApproval(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRevertApproval {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.RevertApproval(c.Request.Context(), id); err != nil {
		utils.SetErrorMessage(c, "Error al revertir aprobación: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Estado revertido a SOLICITADO")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) Reject(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanApproveReject {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.Reject(c.Request.Context(), id); err != nil {
		utils.SetErrorMessage(c, "Error al rechazar la solicitud: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Solicitud RECHAZADA")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) ApproveItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	perms := solicitud.Permissions
	if !perms.CanApproveReject {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.ApproveItem(c.Request.Context(), id, itemID); err != nil {
		utils.SetErrorMessage(c, "Error al aprobar el tramo: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Tramo APROBADO correctamente")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) RejectItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	perms := solicitud.Permissions
	if !perms.CanApproveReject {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.RejectItem(c.Request.Context(), id, itemID); err != nil {
		utils.SetErrorMessage(c, "Error al rechazar el tramo: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Tramo RECHAZADO")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) RevertApprovalItem(c *gin.Context) {
	id := c.Param("id")
	itemID := c.Param("item_id")
	authUser := appcontext.AuthUser(c)
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanRevertApproval {
		utils.SetErrorMessage(c, "No tiene permisos para realizar esta acción")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if err := ctrl.solicitudService.RevertApprovalItem(c.Request.Context(), id, itemID); err != nil {
		utils.SetErrorMessage(c, "Error al revertir aprobación del tramo: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}
	utils.SetSuccessMessage(c, "Aprobación de tramo REVERTIDA")
	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}

func (ctrl *SolicitudOficialController) Print(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error al obtener la solicitud: "+err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		utils.Render(c, "solicitud/oficial/modal_print", gin.H{
			"Solicitud": solicitud,
		})
		return
	}

	personaView, _ := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), solicitud.Usuario.CI)
	pdf := ctrl.reportService.GeneratePV02(c.Request.Context(), solicitud, personaView)
	disposition := "inline"
	if utils.IsMobileBrowser(c) {
		disposition = "attachment"
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"FORM-PV02-%s.pdf\"", disposition, solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudOficialController) GetEditModal(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	authUser := appcontext.AuthUser(c)
	solicitud.HydratePermissions(authUser)
	if !solicitud.Permissions.CanEdit {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	ambitos, _ := ctrl.ambitoService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	tipos, _ := ctrl.tipoSolicitudService.GetByConcepto(c.Request.Context(), "OFICIAL")

	// Items are now pre-sorted by the repository (tipo, fecha, created_at)
	sortSlicePlaceholder := 0 // Ensuring manual removal of the next lines
	_ = sortSlicePlaceholder

	type tramoInicial struct {
		ID           string `json:"id"`
		Tipo         string `json:"tipo"`
		OrigenIATA   string `json:"origen"`
		OrigenLabel  string `json:"origenLabel"`
		DestinoIATA  string `json:"destino"`
		DestinoLabel string `json:"destinoLabel"`
		FechaSalida  string `json:"fechaSalida"`
		AerolineaID  string `json:"aerolinea_id"`
		Estado       string `json:"estado"`
		CanEdit      bool   `json:"can_edit"`
	}

	var tramosIniciales []tramoInicial
	for _, item := range solicitud.Items {
		item.HydratePermissions(authUser)
		tipo := "IDA"
		if item.Tipo == models.TipoSolicitudItemVuelta {
			tipo = "VUELTA"
		}
		origenLabel := item.GetOrigenLabel()
		destinoLabel := item.GetDestinoLabel()
		fechaSalida := ""
		if item.Fecha != nil {
			fechaSalida = item.Fecha.Format("2006-01-02T15:04")
		}
		aerolineaID := ""
		if item.AerolineaID != nil {
			aerolineaID = *item.AerolineaID
		}
		tramosIniciales = append(tramosIniciales, tramoInicial{
			ID:           item.ID,
			Tipo:         tipo,
			OrigenIATA:   item.OrigenIATA,
			OrigenLabel:  origenLabel,
			DestinoIATA:  item.DestinoIATA,
			DestinoLabel: destinoLabel,
			FechaSalida:  fechaSalida,
			AerolineaID:  aerolineaID,
			Estado:       item.GetEstado(),
			CanEdit:      item.Permissions.CanEdit,
		})
	}

	tramosJSON, _ := json.Marshal(tramosIniciales)
	editFormData, _ := json.Marshal(map[string]any{
		"ambito":       solicitud.AmbitoViajeCodigo,
		"tipo":         solicitud.TipoSolicitudCodigo,
		"motivo":       solicitud.Motivo,
		"autorizacion": solicitud.Autorizacion,
		"aerolinea_id": solicitud.AerolineaID,
	})

	destinosPayload := make([]map[string]string, 0, len(destinos))
	for _, d := range destinos {
		destinosPayload = append(destinosPayload, map[string]string{
			"value":  d.IATA,
			"label":  d.GetNombreLargo(),
			"ambito": d.AmbitoCodigo,
		})
	}
	destinosJSON, _ := json.Marshal(destinosPayload)

	utils.Render(c, "solicitud/oficial/modal_edit", gin.H{
		"Solicitud":    solicitud,
		"Aerolineas":   aerolineas,
		"Ambitos":      ambitos,
		"Destinos":     destinos,
		"Tipos":        tipos,
		"TramosJSON":   template.JS(tramosJSON),
		"EditFormData": template.JS(editFormData),
		"DestinosJSON": template.JS(destinosJSON),
	})
}

func (ctrl *SolicitudOficialController) Update(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		utils.SetErrorMessage(c, "Solicitud no encontrada")
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	if !solicitud.CanEdit(authUser) {
		utils.SetErrorMessage(c, "No tiene permisos para editar esta solicitud")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	var req dtos.CreateSolicitudOficialRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
		return
	}

	if js := c.PostForm("tramos_ida_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.TramosIda = tramos
		}
	}
	if js := c.PostForm("tramos_vuelta_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.TramosVuelta = tramos
		}
	}

	if err := ctrl.solicitudOficialService.UpdateOficial(c.Request.Context(), id, req); err != nil {
		utils.SetErrorMessage(c, "Error al actualizar: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Solicitud actualizada correctamente")
	}

	// If HTMX request, re-render the edit modal with updated data
	if c.GetHeader("HX-Request") == "true" {
		ctrl.GetEditModal(c)
		return
	}

	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}
