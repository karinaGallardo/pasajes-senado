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
	"sort"

	"html/template"

	"github.com/gin-gonic/gin"
)

type SolicitudOficialController struct {
	solicitudService      *services.SolicitudService
	destinoService        *services.DestinoService
	tipoSolicitudService  *services.TipoSolicitudService
	ambitoService         *services.AmbitoService
	userService           *services.UsuarioService
	tipoItinerarioService *services.TipoItinerarioService
	aerolineaService      *services.AerolineaService
	reportService         *services.ReportService
	peopleService         *services.PeopleService
	descargoService       *services.DescargoService
}

func NewSolicitudOficialController() *SolicitudOficialController {
	return &SolicitudOficialController{
		solicitudService:      services.NewSolicitudService(),
		destinoService:        services.NewDestinoService(),
		tipoSolicitudService:  services.NewTipoSolicitudService(),
		ambitoService:         services.NewAmbitoService(),
		userService:           services.NewUsuarioService(),
		tipoItinerarioService: services.NewTipoItinerarioService(),
		aerolineaService:      services.NewAerolineaService(),
		reportService:         services.NewReportService(),
		peopleService:         services.NewPeopleService(),
		descargoService:       services.NewDescargoService(),
	}
}

func (ctrl *SolicitudOficialController) GetCreateModal(c *gin.Context) {
	authUser := appcontext.AuthUser(c)

	// Fetch necessary data for the form
	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	ambitos, _ := ctrl.ambitoService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

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
		"AuthUser":    authUser,
		"TargetUser":  targetUser,
		"Aerolineas":  aerolineas,
		"Ambitos":     ambitos,
		"Destinos":    destinos,
		"IsAdmin":     authUser.IsAdminOrResponsable() || (targetUser != nil && authUser.ID != targetUser.ID),
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

	if js := c.PostForm("tramos_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.Tramos = tramos
		}
	}

	_, err := ctrl.solicitudService.CreateOficial(c.Request.Context(), req, authUser)
	if err != nil {
		utils.SetErrorMessage(c, "Error al crear la solicitud: "+err.Error())
		c.Redirect(http.StatusFound, "/solicitudes")
		return
	}

	utils.SetSuccessMessage(c, "Solicitud de Comisión Oficial creada correctamente")
	c.Redirect(http.StatusFound, "/solicitudes")
}

func (ctrl *SolicitudOficialController) Show(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)

	solicitud, err := ctrl.solicitudService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.HTML(http.StatusNotFound, "errors/404", gin.H{"Title": "Solicitud no encontrada"})
		return
	}

	canView := false
	if authUser.IsAdminOrResponsable() || solicitud.UsuarioID == authUser.ID {
		canView = true
	} else if solicitud.Usuario.EncargadoID != nil && *solicitud.Usuario.EncargadoID == authUser.ID {
		canView = true
	} else if solicitud.CreatedBy != nil && *solicitud.CreatedBy == authUser.ID {
		canView = true
	}

	if !canView {
		c.HTML(http.StatusForbidden, "errors/403", gin.H{"Title": "No autorizado"})
		return
	}

	// --- 0. Sort Items (IDA first, then VUELTA) ---
	sort.Slice(solicitud.Items, func(i, j int) bool {
		ti := solicitud.Items[i].Tipo
		tj := solicitud.Items[j].Tipo

		if ti == models.TipoSolicitudItemIda && tj == models.TipoSolicitudItemVuelta {
			return true
		}
		if ti == models.TipoSolicitudItemVuelta && tj == models.TipoSolicitudItemIda {
			return false
		}

		// Chronological fallback if same type or unknown
		if solicitud.Items[i].Fecha != nil && solicitud.Items[j].Fecha != nil {
			return solicitud.Items[i].Fecha.Before(*solicitud.Items[j].Fecha)
		}
		return false
	})

	st := "SOLICITADO"
	if solicitud.EstadoSolicitudCodigo != nil {
		st = *solicitud.EstadoSolicitudCodigo
	}

	// --- 1. Permissions Logic ---
	hasEmitted := false
	for _, item := range solicitud.Items {
		for _, p := range item.Pasajes {
			if p.EstadoPasajeCodigo != nil && *p.EstadoPasajeCodigo == "EMITIDO" {
				hasEmitted = true
				break
			}
		}
		if hasEmitted {
			break
		}
	}

	perms := SolicitudPermissions{
		CanEdit:           authUser.CanEditSolicitud(*solicitud),
		CanApproveReject:  authUser.CanApproveReject(),
		CanRevertApproval: authUser.IsAdminOrResponsable() && (st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO") && !hasEmitted,
		CanAssignPasaje:   authUser.IsAdminOrResponsable(),
		CanAssignViatico:  authUser.IsAdminOrResponsable() && (st == "APROBADO" || st == "PARCIALMENTE_APROBADO" || st == "EMITIDO" || st == "FINALIZADO"),
		CanMakeDescargo:   hasEmitted,
		IsAdminOrResp:     authUser.IsAdminOrResponsable(),
	}

	approvalLabel := "Acciones"

	// --- 2. Stepper Logic ---
	makeStep := func(active, completed bool, colorBase, icon, label string) StepView {
		sv := StepView{
			Icon:  icon,
			Label: label,
		}
		if active || completed {
			sv.WrapperClass = fmt.Sprintf("bg-%s-500 text-white border-none", colorBase)
			sv.LabelClass = fmt.Sprintf("text-%s-500", colorBase)
		} else {
			sv.WrapperClass = "bg-white border-2 border-neutral-200 text-neutral-400"
			sv.LabelClass = "text-neutral-400"
		}
		return sv
	}

	steps := make(map[string]StepView)
	steps["Solicitado"] = makeStep(true, true, "primary", "ph ph-file-text text-lg", "Solicitado")

	rejected := st == "RECHAZADO"
	parcial := st == "PARCIALMENTE_APROBADO"

	if rejected {
		steps["Aprobado"] = StepView{
			Icon:         "ph ph-x text-xl font-bold",
			Label:        "Rechazado",
			WrapperClass: "bg-danger-500 text-white border-none",
			LabelClass:   "text-danger-500",
		}
	} else if parcial {
		steps["Aprobado"] = StepView{
			Icon:         "ph ph-check-square-offset text-xl",
			Label:        "Parcial",
			WrapperClass: "bg-violet-500 text-white border-none",
			LabelClass:   "text-violet-500",
		}
	} else {
		isAp := st == "APROBADO" || st == "EMITIDO" || st == "FINALIZADO"
		steps["Aprobado"] = makeStep(isAp, isAp, "success", "ph ph-check text-xl font-bold", "Aprobado")
	}

	isEm := st == "EMITIDO" || st == "FINALIZADO"
	steps["Emitido"] = makeStep(isEm, isEm, "secondary", "ph ph-ticket text-xl", "Emitido")

	isFin := st == "FINALIZADO"
	steps["Finalizado"] = makeStep(isFin, isFin, "neutral", "ph ph-flag-checkered text-xl", "Finalizado")

	showNextSteps := !rejected

	// --- 3. Status Card Logic ---
	statusCard := StatusCardView{}
	switch st {
	case "SOLICITADO":
		statusCard.BorderClass = "border-primary-500"
		statusCard.TextClass = "text-primary-700"
	case "RECHAZADO":
		statusCard.BorderClass = "border-danger-500"
		statusCard.TextClass = "text-danger-700"
	case "APROBADO":
		statusCard.BorderClass = "border-success-500"
		statusCard.TextClass = "text-success-700"
	case "PARCIALMENTE_APROBADO":
		statusCard.BorderClass = "border-violet-500"
		statusCard.TextClass = "text-violet-700"
	case "EMITIDO":
		statusCard.BorderClass = "border-secondary-500"
		statusCard.TextClass = "text-secondary-700"
	case "FINALIZADO":
		statusCard.BorderClass = "border-neutral-500"
		statusCard.TextClass = "text-neutral-700"
	default:
		statusCard.BorderClass = "border-neutral-200"
		statusCard.TextClass = "text-neutral-900"
	}

	// --- 4. Pasajes Views ---
	var pasajesViews []PasajeView
	for _, item := range solicitud.Items {
		for i := range item.Pasajes {
			p := &item.Pasajes[i]
			pCode := ""
			if p.EstadoPasajeCodigo != nil {
				pCode = *p.EstadoPasajeCodigo
			}

			pv := PasajeView{Pasaje: *p}
			if p.EstadoPasaje != nil {
				pv.StatusColorClass = fmt.Sprintf("bg-%s-100 text-%s-800", p.EstadoPasaje.Color, p.EstadoPasaje.Color)
			} else {
				switch pCode {
				case "REGISTRADO":
					pv.StatusColorClass = "bg-secondary-100 text-secondary-800"
				case "EMITIDO":
					pv.StatusColorClass = "bg-success-100 text-success-800"
				case "USADO":
					pv.StatusColorClass = "bg-primary-100 text-primary-800"
				case "ANULADO":
					pv.StatusColorClass = "bg-neutral-100 text-neutral-800"
				default:
					pv.StatusColorClass = "bg-neutral-100 text-neutral-800"
				}
			}

			pPerms := PasajePermissions{}
			if item.GetEstado() != "REPROGRAMADO" {
				pPerms.CanEdit = authUser.IsAdminOrResponsable() && pCode == "REGISTRADO"
				pPerms.CanMarkUsado = authUser.CanMarkUsado(*solicitud) && pCode == "EMITIDO"
				pPerms.CanDevolver = authUser.IsAdminOrResponsable() && pCode == "EMITIDO"
				pPerms.CanAnular = authUser.IsAdminOrResponsable() && (pCode == "REGISTRADO" || pCode == "EMITIDO")
				pPerms.CanEmitir = authUser.IsAdminOrResponsable() && pCode == "REGISTRADO"
				pPerms.ShowActionsMenu = pPerms.CanEdit || pPerms.CanMarkUsado || pPerms.CanDevolver || pPerms.CanAnular || pPerms.CanEmitir
			}
			pv.Perms = pPerms
			pasajesViews = append(pasajesViews, pv)
		}
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	// Descargo PV-05/06: si ya existe, pasamos ID y Estado para enlaces directos
	var descargoID string
	var descargoEstado string
	if descargo, _ := ctrl.descargoService.GetBySolicitudID(c.Request.Context(), id); descargo != nil && descargo.ID != "" {
		descargoID = descargo.ID
		descargoEstado = descargo.Estado
	}

	utils.Render(c, "solicitud/oficial/show", gin.H{
		"Title":          "Solicitud de Comisión Oficial " + solicitud.Codigo,
		"Solicitud":      solicitud,
		"AuthUser":       authUser,
		"Perms":          perms,
		"Steps":          steps,
		"ShowNextSteps":  showNextSteps,
		"StatusCard":     statusCard,
		"PasajesView":    pasajesViews,
		"ApprovalLabel":  approvalLabel,
		"Aerolineas":     aerolineas,
		"DescargoID":     descargoID,
		"DescargoEstado": descargoEstado,
	})
}

func (ctrl *SolicitudOficialController) Approve(c *gin.Context) {
	id := c.Param("id")
	authUser := appcontext.AuthUser(c)
	if authUser == nil || !authUser.CanApproveReject() {
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
	if authUser == nil || !authUser.IsAdminOrResponsable() {
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
	if authUser == nil || !authUser.CanApproveReject() {
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
	if authUser == nil || !authUser.CanApproveReject() {
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
	if authUser == nil || !authUser.CanApproveReject() {
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
	if authUser == nil || !authUser.IsAdminOrResponsable() {
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
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV02-%s.pdf", solicitud.Codigo))
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
	if !authUser.CanEditSolicitud(*solicitud) {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())
	ambitos, _ := ctrl.ambitoService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())

	// Ordenar ítems: IDA primero, luego VUELTA, por fecha
	sort.Slice(solicitud.Items, func(i, j int) bool {
		if solicitud.Items[i].Tipo != solicitud.Items[j].Tipo {
			return solicitud.Items[i].Tipo == models.TipoSolicitudItemIda
		}
		if solicitud.Items[i].Fecha != nil && solicitud.Items[j].Fecha != nil {
			return solicitud.Items[i].Fecha.Before(*solicitud.Items[j].Fecha)
		}
		return i < j
	})

	type tramoInicial struct {
		ID           string `json:"id"`
		Tipo         string `json:"tipo"`
		OrigenIATA   string `json:"origen"`
		OrigenLabel  string `json:"origenLabel"`
		DestinoIATA  string `json:"destino"`
		DestinoLabel string `json:"destinoLabel"`
		FechaSalida  string `json:"fechaSalida"`
		Estado       string `json:"estado"`
	}

	var tramosIniciales []tramoInicial
	for _, item := range solicitud.Items {
		tipo := "IDA"
		if item.Tipo == models.TipoSolicitudItemVuelta {
			tipo = "VUELTA"
		}
		origenLabel := item.OrigenIATA
		if item.Origen != nil {
			origenLabel = item.OrigenIATA + " - " + item.Origen.GetLabel()
		}
		destinoLabel := item.DestinoIATA
		if item.Destino != nil {
			destinoLabel = item.DestinoIATA + " - " + item.Destino.GetLabel()
		}
		fechaSalida := ""
		if item.Fecha != nil {
			fechaSalida = item.Fecha.Format("2006-01-02T15:04")
		}
		tramosIniciales = append(tramosIniciales, tramoInicial{
			ID:           item.ID,
			Tipo:         tipo,
			OrigenIATA:   item.OrigenIATA,
			OrigenLabel:  origenLabel,
			DestinoIATA:  item.DestinoIATA,
			DestinoLabel: destinoLabel,
			FechaSalida:  fechaSalida,
			Estado:       item.GetEstado(),
		})
	}

	tramosJSON, _ := json.Marshal(tramosIniciales)
	editFormData, _ := json.Marshal(map[string]string{
		"ambito":       solicitud.AmbitoViajeCodigo,
		"motivo":       solicitud.Motivo,
		"autorizacion": solicitud.Autorizacion,
		"aerolinea":    solicitud.AerolineaSugerida,
	})

	destinosPayload := make([]map[string]string, 0, len(destinos))
	for _, d := range destinos {
		destinosPayload = append(destinosPayload, map[string]string{
			"value":  d.IATA,
			"label":  d.GetLabel(),
			"ambito": d.AmbitoCodigo,
		})
	}
	destinosJSON, _ := json.Marshal(destinosPayload)

	utils.Render(c, "solicitud/oficial/modal_edit", gin.H{
		"Solicitud":    solicitud,
		"Aerolineas":   aerolineas,
		"Ambitos":      ambitos,
		"Destinos":     destinos,
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

	if !authUser.CanEditSolicitud(*solicitud) {
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

	if js := c.PostForm("tramos_json"); js != "" {
		var tramos []dtos.TramoOficialRequest
		if err := json.Unmarshal([]byte(js), &tramos); err == nil {
			req.Tramos = tramos
		}
	}

	if err := ctrl.solicitudService.UpdateOficial(c.Request.Context(), id, req); err != nil {
		utils.SetErrorMessage(c, "Error al actualizar: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Solicitud actualizada correctamente")
	}

	c.Redirect(http.StatusFound, "/solicitudes/oficial/"+id+"/detalle")
}
