package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	service               *services.SolicitudService
	destinoService        *services.DestinoService
	conceptoService       *services.ConceptoService
	tipoSolicitudService  *services.TipoSolicitudService
	ambitoService         *services.AmbitoService
	tipoItinerarioService *services.TipoItinerarioService
	cupoService           *services.CupoService
	userService           *services.UsuarioService
	peopleService         *services.PeopleService
	reportService         *services.ReportService
	aerolineaService      *services.AerolineaService
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:               services.NewSolicitudService(),
		destinoService:        services.NewDestinoService(),
		conceptoService:       services.NewConceptoService(),
		tipoSolicitudService:  services.NewTipoSolicitudService(),
		ambitoService:         services.NewAmbitoService(),
		tipoItinerarioService: services.NewTipoItinerarioService(),
		cupoService:           services.NewCupoService(),
		userService:           services.NewUsuarioService(),
		peopleService:         services.NewPeopleService(),
		reportService:         services.NewReportService(),
		aerolineaService:      services.NewAerolineaService(),
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.service.GetAll(c.Request.Context())

	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}

	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}

	usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
	usuariosMap := make(map[string]*models.Usuario)
	for i := range usuarios {
		usuariosMap[usuarios[i].ID] = &usuarios[i]
	}

	utils.Render(c, "solicitud/index", gin.H{
		"Title":       "Bandeja de Solicitudes",
		"Solicitudes": solicitudes,
		"Usuarios":    usuariosMap,
	})
}

func (ctrl *SolicitudController) Create(c *gin.Context) {
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	conceptos, _ := ctrl.conceptoService.GetAll(c.Request.Context())

	currentUser := appcontext.CurrentUser(c)
	targetUserID := c.Query("user_id")
	var targetUser *models.Usuario

	if targetUserID != "" {
		u, err := ctrl.userService.GetByID(c.Request.Context(), targetUserID)
		if err == nil {
			targetUser = u
		} else {
			targetUser = currentUser
		}
	} else {
		targetUser = currentUser
	}

	var alertaOrigen string
	if targetUser.GetOrigenIATA() == "" {
		alertaOrigen = "Este usuario no tiene configurado su LUGAR DE ORIGEN en el perfil. El sistema no podrá calcular rutas automáticamente."
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	now := time.Now()
	vouchers, _ := ctrl.cupoService.GetVouchersByUsuario(c.Request.Context(), targetUser.ID, now.Year(), int(now.Month()))

	utils.Render(c, "solicitud/create", gin.H{
		"Title":        "Nueva Solicitud de Pasaje",
		"TargetUser":   targetUser,
		"Destinos":     destinos,
		"Conceptos":    conceptos,
		"Aerolineas":   aerolineas,
		"AlertaOrigen": alertaOrigen,
		"Vouchers":     vouchers,
		"ItinerarioIdaID": func() string {
			itin, _ := ctrl.tipoItinerarioService.GetByCodigo(c.Request.Context(), "SOLO_IDA")
			if itin != nil {
				return itin.ID
			}
			return ""
		}(),
		"ItinerarioVueltaID": func() string {
			itin, _ := ctrl.tipoItinerarioService.GetByCodigo(c.Request.Context(), "SOLO_VUELTA")
			if itin != nil {
				return itin.ID
			}
			return ""
		}(),
	})
}

func (ctrl *SolicitudController) CheckCupo(c *gin.Context) {
	usuario := appcontext.CurrentUser(c)
	if usuario == nil {
		c.String(http.StatusUnauthorized, "No autorizado")
		return
	}

	targetUserID := c.Query("target_user_id")
	if targetUserID == "" {
		targetUserID = c.Query("user_id")
	}
	if targetUserID == "" {
		targetUserID = usuario.ID
	}

	tipoID := c.Query("tipo_solicitud_id")
	if tipoID == "" {
		c.String(http.StatusOK, "")
		return
	}

	tipo, err := ctrl.tipoSolicitudService.GetByID(c.Request.Context(), tipoID)
	if err != nil || tipo.ConceptoViaje == nil || tipo.ConceptoViaje.Codigo != "DERECHO" {
		c.String(http.StatusOK, "")
		return
	}

	fecha := time.Now()
	fechaStr := c.Query("fecha_salida")
	if fechaStr != "" {
		layout := "2006-01-02T15:04"
		if parsed, err := time.Parse(layout, fechaStr); err == nil {
			fecha = parsed
		}
	}

	info, err := ctrl.cupoService.CalcularCupo(c.Request.Context(), targetUserID, fecha)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error calculando cupo")
		return
	}

	colorClass := "bg-senado-50 text-senado-900 border-senado-200"
	if !info.EsDisponible {
		colorClass = "bg-red-50 text-red-800 border-red-200"
	}

	var statusIcon string
	if info.EsDisponible {
		statusIcon = `<svg class="h-4 w-4 text-green-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`
	} else {
		statusIcon = `<svg class="h-4 w-4 text-red-500 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>`
	}

	html := fmt.Sprintf(`
		<div class="flex items-center p-2 rounded border %s">
			%s
			<div class="flex-1">
				<p class="text-sm font-bold leading-tight">%s</p>
			</div>
		</div>
	`, colorClass, statusIcon, info.Mensaje)

	c.Writer.Header().Set("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func (ctrl *SolicitudController) Store(c *gin.Context) {
	var req dtos.CreateSolicitudRequest
	if err := c.ShouldBind(&req); err != nil {
		destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
		utils.Render(c, "solicitud/create", gin.H{
			"Error":    "Datos inválidos: " + err.Error(),
			"Destinos": destinos,
		})
		return
	}

	usuario := appcontext.CurrentUser(c)
	if usuario == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	solicitud, err := ctrl.service.Create(c.Request.Context(), req, usuario)
	if err != nil {
		destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
		utils.Render(c, "solicitud/create", gin.H{
			"Error":    "Error: " + err.Error(),
			"Destinos": destinos,
		})
		return
	}

	if req.VoucherID != "" {
		c.Redirect(http.StatusFound, fmt.Sprintf("/cupos/derecho/%s", solicitud.UsuarioID))
		return
	}

	path := c.Request.Header.Get("Referer")
	if path == "" {
		path = "/solicitudes"
	}
	c.Redirect(http.StatusFound, path)
}

func (ctrl *SolicitudController) Show(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	st := "SOLICITADO"
	if solicitud.EstadoSolicitudCodigo != nil {
		st = *solicitud.EstadoSolicitudCodigo
	}
	step1 := true
	step2 := st == "APROBADO" || st == "FINALIZADO"
	step3 := st == "FINALIZADO"

	mermaidGraph := "graph TD; A[Registro Solicitud] --> B{¿Autorización?}; B -- Aprobado --> C[Gestión Pasajes]; C --> D[Viaje / Finalizado]; B -- Rechazado --> E[Solicitud Rechazada];\n"
	mermaidGraph += "classDef default fill:#fff,stroke:#333,stroke-width:1px; classDef active fill:#03738C,stroke:#03738C,stroke-width:2px,color:#fff;\n"

	switch st {
	case "SOLICITADO":
		mermaidGraph += "class A active;"
	case "APROBADO":
		mermaidGraph += "class C active;"
	case "FINALIZADO":
		mermaidGraph += "class D active;"
	case "RECHAZADO":
		mermaidGraph += "class E active;"
	}

	aerolineas, _ := ctrl.aerolineaService.GetAllActive(c.Request.Context())

	userIDsMap := make(map[string]bool)
	if solicitud.CreatedBy != nil {
		userIDsMap[*solicitud.CreatedBy] = true
	}
	if solicitud.UpdatedBy != nil {
		userIDsMap[*solicitud.UpdatedBy] = true
	}

	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}

	usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
	usuariosMap := make(map[string]*models.Usuario)
	for i := range usuarios {
		usuariosMap[usuarios[i].ID] = &usuarios[i]
	}

	utils.Render(c, "solicitud/show", gin.H{
		"Title":        "Detalle Solicitud #" + id,
		"Solicitud":    solicitud,
		"Usuarios":     usuariosMap,
		"Step1":        step1,
		"Step2":        step2,
		"Step3":        step3,
		"MermaidGraph": mermaidGraph,
		"Aerolineas":   aerolineas,
		"User":         appcontext.CurrentUser(c),
	})
}

func (ctrl *SolicitudController) PrintPV01(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		utils.Render(c, "solicitud/derecho/modal_print", gin.H{
			"Solicitud": solicitud,
		})
		return
	}

	personaView, errMongo := ctrl.peopleService.GetSenatorDataByCI(c.Request.Context(), solicitud.Usuario.CI)
	if errMongo != nil {
		personaView = nil
	}

	pdf := ctrl.reportService.GeneratePV01(c.Request.Context(), solicitud, personaView)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV01-%s.pdf", solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudController) Approve(c *gin.Context) {
	id := c.Param("id")
	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil || !currentUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.service.Approve(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error al aprobar la solicitud: "+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}

func (ctrl *SolicitudController) Reject(c *gin.Context) {
	id := c.Param("id")
	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil || !currentUser.CanApproveReject() {
		c.String(http.StatusForbidden, "No tiene permisos para realizar esta acción")
		return
	}
	if err := ctrl.service.Reject(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "Error al rechazar la solicitud: "+err.Error())
		return
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}

func (ctrl *SolicitudController) Edit(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No se puede editar una solicitud que no está en estado SOLICITADO")
		return
	}

	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if !currentUser.CanEditSolicitud(*solicitud) {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	if solicitud.VoucherID != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/solicitudes/derecho/%s/editar", id))
		return
	}

	tipos, _ := ctrl.tipoSolicitudService.GetAll(c.Request.Context())
	ambitos, _ := ctrl.ambitoService.GetAll(c.Request.Context())
	destinos, _ := ctrl.destinoService.GetAll(c.Request.Context())
	tiposItinerario, _ := ctrl.tipoItinerarioService.GetAll(c.Request.Context())

	utils.Render(c, "solicitud/edit", gin.H{
		"Title":           "Editar Solicitud",
		"Solicitud":       solicitud,
		"TiposSolicitud":  tipos,
		"AmbitosViaje":    ambitos,
		"Destinos":        destinos,
		"TiposItinerario": tiposItinerario,
	})
}

func (ctrl *SolicitudController) Update(c *gin.Context) {
	id := c.Param("id")
	var req dtos.UpdateSolicitudRequest

	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	} else {
		c.String(http.StatusBadRequest, "Formato fecha salida inválido")
		return
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" {
		if t, err := time.Parse(layout, req.FechaVuelta); err == nil {
			fechaVuelta = &t
		} else {
			c.String(http.StatusBadRequest, "Formato fecha retorno inválido")
			return
		}
	}

	solicitud, err := ctrl.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.EstadoSolicitudCodigo != nil && *solicitud.EstadoSolicitudCodigo != "SOLICITADO" {
		c.String(http.StatusForbidden, "No editable")
		return
	}

	solicitud.TipoSolicitudID = req.TipoSolicitudID
	solicitud.AmbitoViajeID = req.AmbitoViajeID
	if req.TipoItinerarioID != "" {
		solicitud.TipoItinerarioID = req.TipoItinerarioID
	}
	solicitud.OrigenIATA = req.OrigenIATA
	solicitud.DestinoIATA = req.DestinoIATA
	solicitud.FechaIda = fechaIda
	solicitud.FechaVuelta = fechaVuelta
	solicitud.Motivo = req.Motivo
	solicitud.AerolineaSugerida = req.AerolineaSugerida
	if err := ctrl.service.Update(c.Request.Context(), solicitud); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando: "+err.Error())
		return
	}

	if solicitud.VoucherID != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("/cupos/derecho/%s", solicitud.UsuarioID))
		return
	}

	path := c.Request.Header.Get("Referer")
	if path == "" {
		path = "/solicitudes/" + id
	}

	c.Redirect(http.StatusFound, path)
}
