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
	service          *services.SolicitudService
	ciudadService    *services.CiudadService
	catalogoService  *services.CatalogoService
	cupoService      *services.CupoService
	userService      *services.UsuarioService
	peopleService    *services.PeopleService
	reportService    *services.ReportService
	aerolineaService *services.AerolineaService
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:          services.NewSolicitudService(),
		ciudadService:    services.NewCiudadService(),
		catalogoService:  services.NewCatalogoService(),
		cupoService:      services.NewCupoService(),
		userService:      services.NewUsuarioService(),
		peopleService:    services.NewPeopleService(),
		reportService:    services.NewReportService(),
		aerolineaService: services.NewAerolineaService(),
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	solicitudes, _ := ctrl.service.FindAll()
	utils.Render(c, "solicitud/index.html", gin.H{
		"Title":       "Bandeja de Solicitudes",
		"Solicitudes": solicitudes,
	})
}

func (ctrl *SolicitudController) Create(c *gin.Context) {
	destinos, _ := ctrl.ciudadService.GetAll()
	conceptos, _ := ctrl.catalogoService.GetAllConceptos()

	currentUser := appcontext.CurrentUser(c)
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

	aereos, _ := ctrl.aerolineaService.GetAllActive()
	var aerolineas []string
	for _, a := range aereos {
		aerolineas = append(aerolineas, a.Nombre)
	}

	now := time.Now()
	vouchers, _ := ctrl.cupoService.GetVouchersByUsuario(targetUser.ID, now.Year(), int(now.Month()))

	utils.Render(c, "solicitud/create.html", gin.H{
		"Title":        "Nueva Solicitud de Pasaje",
		"TargetUser":   targetUser,
		"Destinos":     destinos,
		"Conceptos":    conceptos,
		"Aerolineas":   aerolineas,
		"AlertaOrigen": alertaOrigen,
		"Vouchers":     vouchers,
	})
}

func (ctrl *SolicitudController) CreateDerecho(c *gin.Context) {
	destinos, _ := ctrl.ciudadService.GetAll()
	conceptos, _ := ctrl.catalogoService.GetAllConceptos()

	currentUser := appcontext.CurrentUser(c)
	targetUserParamID := c.Param("id")
	var targetUser *models.Usuario

	if targetUserParamID != "" && targetUserParamID != "me" {
		u, err := ctrl.userService.GetByID(targetUserParamID)
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

	aereos, _ := ctrl.aerolineaService.GetAllActive()
	var aerolineas []string
	for _, a := range aereos {
		aerolineas = append(aerolineas, a.Nombre)
	}
	now := time.Now()
	vouchers, _ := ctrl.cupoService.GetVouchersByUsuario(targetUser.ID, now.Year(), int(now.Month()))

	conceptoDer, _ := ctrl.catalogoService.GetConceptoByCodigo("DERECHO")
	tipoCupo, _ := ctrl.catalogoService.GetTipoSolicitudByCodigo("USO_CUPO")
	ambitoNac, _ := ctrl.catalogoService.GetAmbitoByCodigo("NACIONAL")

	utils.Render(c, "solicitud/create.html", gin.H{
		"Title":                  "Pasaje por Derecho - " + targetUser.GetNombreCompleto(),
		"TargetUser":             targetUser,
		"Destinos":               destinos,
		"Conceptos":              conceptos,
		"Aerolineas":             aerolineas,
		"AlertaOrigen":           alertaOrigen,
		"Vouchers":               vouchers,
		"DefaultConceptoID":      getObjectID(conceptoDer),
		"DefaultTipoSolicitudID": getObjectID(tipoCupo),
		"DefaultAmbitoID":        getObjectID(ambitoNac),
	})
}

func getObjectID(obj interface{}) string {
	if obj == nil {
		return ""
	}
	switch v := obj.(type) {
	case *models.ConceptoViaje:
		return v.ID
	case *models.TipoSolicitud:
		return v.ID
	case *models.AmbitoViaje:
		return v.ID
	}
	return ""
}

func (ctrl *SolicitudController) CheckCupo(c *gin.Context) {
	usuario := appcontext.CurrentUser(c)
	if usuario == nil {
		c.String(http.StatusUnauthorized, "No autorizado")
		return
	}

	targetUserID := c.Query("target_user_id")
	if targetUserID == "" {
		targetUserID = usuario.ID
	}

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

	info, err := ctrl.cupoService.CalcularCupo(targetUserID, fecha)
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
		destinos, _ := ctrl.ciudadService.GetAll()
		utils.Render(c, "solicitud/create.html", gin.H{
			"Error":    "Datos inválidos: " + err.Error(),
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

	usuario := appcontext.CurrentUser(c)
	if usuario == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

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

		utils.Render(c, "solicitud/create.html", gin.H{
			"Error":    "Error: " + err.Error(),
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

	utils.Render(c, "solicitud/show.html", gin.H{
		"Title":        "Detalle Solicitud #" + id,
		"Solicitud":    solicitud,
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
	if errMongo != nil {
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

	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	if solicitud.UsuarioID != currentUser.ID && currentUser.Rol.Codigo != "ADMIN" {
		c.String(http.StatusForbidden, "No tiene permisos para editar esta solicitud")
		return
	}

	tipos, _ := ctrl.catalogoService.GetTiposSolicitud()
	ambitos, _ := ctrl.catalogoService.GetAmbitosViaje()
	ciudades, _ := ctrl.ciudadService.GetAll()
	tiposItinerario, _ := ctrl.catalogoService.GetTiposItinerario()

	utils.Render(c, "solicitud/edit.html", gin.H{
		"Title":           "Editar Solicitud",
		"Solicitud":       solicitud,
		"TiposSolicitud":  tipos,
		"AmbitosViaje":    ambitos,
		"Ciudades":        ciudades,
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
