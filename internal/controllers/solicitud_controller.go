package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	service         *services.SolicitudService
	ciudadService   *services.CiudadService
	catalogoService *services.CatalogoService
	cupoService     *services.CupoService
	userService     *services.UsuarioService
	peopleService   *services.PeopleService
	reportService   *services.ReportService
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:         services.NewSolicitudService(),
		ciudadService:   services.NewCiudadService(),
		catalogoService: services.NewCatalogoService(),
		cupoService:     services.NewCupoService(),
		userService:     services.NewUsuarioService(),
		peopleService:   services.NewPeopleService(),
		reportService:   services.NewReportService(),
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.service.FindAll()
	c.HTML(http.StatusOK, "solicitud/index.html", gin.H{
		"Title":       "Bandeja de Solicitudes",
		"Solicitudes": solicitudes,
		"User":        c.MustGet("User"),
	})
}

func (ctrl *SolicitudController) Create(c *gin.Context) {
	destinos, _ := ctrl.ciudadService.GetAll()
	conceptos, _ := ctrl.catalogoService.GetAllConceptos()

	userContext, _ := c.Get("User")
	currentUser := userContext.(*models.Usuario)

	targetUserID := c.Query("user_id")
	var targetUser *models.Usuario

	if targetUserID != "" {
		u, err := ctrl.userService.GetByID(targetUserID)
		if err == nil {
			targetUser = u
		} else {
			targetUser = currentUser
		}
	} else {
		targetUser = currentUser
	}

	var alertaOrigen string
	if targetUser.GetOrigenCode() == "" {
		alertaOrigen = "Este usuario no tiene configurado su LUGAR DE ORIGEN en el perfil. El sistema no podrá calcular rutas automáticamente."
	}

	aerolineas := []string{"BoA - Boliviana de Aviación", "EcoJet"}

	c.HTML(http.StatusOK, "solicitud/create.html", gin.H{
		"Title":        "Nueva Solicitud de Pasaje",
		"User":         targetUser,
		"CurrentUser":  currentUser,
		"Destinos":     destinos,
		"Conceptos":    conceptos,
		"Aerolineas":   aerolineas,
		"AlertaOrigen": alertaOrigen,
	})
}

func (ctrl *SolicitudController) CheckCupo(c *gin.Context) {
	userContext, _ := c.Get("User")
	usuario := userContext.(*models.Usuario)

	tipoID := c.Query("tipo_solicitud_id")
	if tipoID == "" {
		c.String(http.StatusOK, "")
		return
	}

	tipo, err := ctrl.catalogoService.GetTipoByID(tipoID)
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

	info, err := ctrl.cupoService.CalcularCupo(usuario.ID, fecha)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error calculando cupo")
		return
	}

	colorClass := "bg-senado-50 text-senado-900 border-senado-200"
	if !info.EsDisponible {
		colorClass = "bg-red-50 text-red-800 border-red-200"
	}

	html := fmt.Sprintf(`
		<div class="border rounded-md p-3 mt-2 %s">
			<label class="block text-xs font-bold uppercase tracking-wide opacity-70 mb-1">*Por Derecho :</label>
			<p class="text-lg font-mono font-bold">%s</p>
		</div>
	`, colorClass, info.Mensaje)

	c.Writer.Header().Set("Content-Type", "text/html")
	c.String(http.StatusOK, html)
}

func (ctrl *SolicitudController) Store(c *gin.Context) {
	var req dtos.CreateSolicitudRequest
	if err := c.ShouldBind(&req); err != nil {
		destinos, _ := ctrl.ciudadService.GetAll()
		c.HTML(http.StatusOK, "solicitud/create.html", gin.H{
			"Error":    "Datos inválidos: " + err.Error(),
			"User":     c.MustGet("User"),
			"Destinos": destinos,
		})
		return
	}

	layout := "2006-01-02T15:04"
	fechaSalida, _ := time.Parse(layout, req.FechaSalida)

	var fechaRetorno time.Time
	if req.FechaRetorno != "" {
		fechaRetorno, _ = time.Parse(layout, req.FechaRetorno)
	}

	userContext, exists := c.Get("User")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	usuario := userContext.(*models.Usuario)

	var realSolicitanteID string
	if req.TargetUserID != "" {
		realSolicitanteID = req.TargetUserID
	} else {
		realSolicitanteID = usuario.ID
	}

	itinCode := req.TipoItinerarioCode
	if itinCode == "" {
		itinCode = "IDA_VUELTA"
	}
	itin, _ := ctrl.catalogoService.GetTipoItinerarioByCodigo(itinCode)
	var itinID string
	if itin != nil {
		itinID = itin.ID
	}

	nuevaSolicitud := models.Solicitud{
		UsuarioID:         realSolicitanteID,
		TipoSolicitudID:   req.TipoSolicitudID,
		AmbitoViajeID:     req.AmbitoViajeID,
		TipoItinerarioID:  itinID,
		OrigenCode:        req.OrigenCode,
		DestinoCode:       req.DestinoCode,
		FechaSalida:       fechaSalida,
		FechaRetorno:      fechaRetorno,
		Motivo:            req.Motivo,
		AerolineaSugerida: req.AerolineaSugerida,
		Estado:            "SOLICITADO",
	}

	if err := ctrl.service.Create(&nuevaSolicitud, usuario); err != nil {
		destinos, _ := ctrl.ciudadService.GetAll()

		c.HTML(http.StatusOK, "solicitud/create.html", gin.H{
			"Error":    "Error: " + err.Error(),
			"User":     c.MustGet("User"),
			"Destinos": destinos,
		})
		return
	}

	c.Redirect(http.StatusFound, "/solicitudes")
}

func (ctrl *SolicitudController) Show(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.FindByID(id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	st := solicitud.Estado
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

	aerolineas := []string{"BoA - Boliviana de Aviación", "EcoJet"}

	c.HTML(http.StatusOK, "solicitud/show.html", gin.H{
		"Title":        "Detalle Solicitud #" + id,
		"Solicitud":    solicitud,
		"User":         c.MustGet("User"),
		"Step1":        step1,
		"Step2":        step2,
		"Step3":        step3,
		"MermaidGraph": mermaidGraph,
		"Aerolineas":   aerolineas,
	})
}

func (ctrl *SolicitudController) PrintPV01(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.FindByID(id)

	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving solicitud: "+err.Error())
		return
	}

	personaView, errMongo := ctrl.peopleService.FindSenatorDataByCI(solicitud.Usuario.CI)
	// We pass personaView even if error, service handles nil
	if errMongo != nil {
		// Log error but proceed? Or maybe personaView is nil.
		// Original logic: if errMongo == nil && personaView != nil
		personaView = nil
	}

	pdf := ctrl.reportService.GeneratePV01(solicitud, personaView)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV01-%s.pdf", solicitud.ID))
	pdf.Output(c.Writer)
}

func (ctrl *SolicitudController) Approve(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.service.Approve(id); err != nil {
		// log.Printf("Error approving solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}

func (ctrl *SolicitudController) Reject(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.service.Reject(id); err != nil {
		// log.Printf("Error rejecting solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}

func (ctrl *SolicitudController) Edit(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.service.FindByID(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.Estado != "SOLICITADO" {
		c.String(http.StatusForbidden, "No se puede editar una solicitud que no está en estado SOLICITADO")
		return
	}

	user, _ := c.Get("User")
	currentUser := user.(*models.Usuario)

	if solicitud.UsuarioID != currentUser.ID && currentUser.Rol.Codigo != "ADMIN" {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	tipos, _ := ctrl.catalogoService.GetTiposSolicitud()
	ambitos, _ := ctrl.catalogoService.GetAmbitosViaje()
	ciudades, _ := ctrl.ciudadService.GetAll()
	tiposItinerario, _ := ctrl.catalogoService.GetTiposItinerario()

	c.HTML(http.StatusOK, "solicitud/edit.html", gin.H{
		"Title":           "Editar Solicitud",
		"Solicitud":       solicitud,
		"TiposSolicitud":  tipos,
		"AmbitosViaje":    ambitos,
		"Ciudades":        ciudades,
		"TiposItinerario": tiposItinerario,
		"User":            currentUser,
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
	fechaSalida, err := time.Parse(layout, req.FechaSalida)
	if err != nil {
		c.String(http.StatusBadRequest, "Formato fecha salida inválido")
		return
	}

	var fechaRetorno time.Time
	if req.FechaRetorno != "" {
		fr, err := time.Parse(layout, req.FechaRetorno)
		if err != nil {
			c.String(http.StatusBadRequest, "Formato fecha retorno inválido")
			return
		}
		fechaRetorno = fr
	}

	solicitud, err := ctrl.service.FindByID(id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	if solicitud.Estado != "SOLICITADO" {
		c.String(http.StatusForbidden, "No editable")
		return
	}

	solicitud.TipoSolicitudID = req.TipoSolicitudID
	solicitud.AmbitoViajeID = req.AmbitoViajeID
	solicitud.TipoItinerarioID = req.TipoItinerarioID
	solicitud.OrigenCode = req.OrigenCod
	solicitud.DestinoCode = req.DestinoCod
	solicitud.FechaSalida = fechaSalida
	solicitud.FechaRetorno = fechaRetorno
	solicitud.Motivo = req.Motivo
	solicitud.AerolineaSugerida = req.AerolineaSugerida

	if err := ctrl.service.Update(solicitud); err != nil {
		c.String(http.StatusInternalServerError, "Error actualizando: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}
