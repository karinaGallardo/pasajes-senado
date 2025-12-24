package controllers

import (
	"fmt"
	"net/http"
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
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:         services.NewSolicitudService(),
		ciudadService:   services.NewCiudadService(),
		catalogoService: services.NewCatalogoService(),
		cupoService:     services.NewCupoService(),
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
	userBasic := userContext.(*models.Usuario)

	var alertaOrigen string
	if userBasic.GetOrigenCode() == "" {
		alertaOrigen = "⚠️ No tiene configurado su LUGAR DE ORIGEN en el perfil. El sistema no podrá calcular rutas automáticamente. Contacte a soporte."
	}

	c.HTML(http.StatusOK, "solicitud/create.html", gin.H{
		"Title":        "Nueva Solicitud de Pasaje",
		"User":         userBasic,
		"Destinos":     destinos,
		"Conceptos":    conceptos,
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
	layout := "2006-01-02T15:04"
	fechaSalida, _ := time.Parse(layout, c.PostForm("fecha_salida"))

	var fechaRetorno time.Time
	if c.PostForm("fecha_retorno") != "" {
		fechaRetorno, _ = time.Parse(layout, c.PostForm("fecha_retorno"))
	}

	userContext, exists := c.Get("User")
	if !exists {
		c.Redirect(http.StatusFound, "/login")
		return
	}
	usuario := userContext.(*models.Usuario)

	itinCode := c.PostForm("tipo_itinerario")
	if itinCode == "" {
		itinCode = "IDA_VUELTA"
	}
	itin, _ := ctrl.catalogoService.GetTipoItinerarioByCodigo(itinCode)
	var itinID string
	if itin != nil {
		itinID = itin.ID
	}

	nuevaSolicitud := models.Solicitud{
		UsuarioID:        usuario.ID,
		TipoSolicitudID:  c.PostForm("tipo_solicitud_id"),
		AmbitoViajeID:    c.PostForm("ambito_viaje_id"),
		TipoItinerarioID: itinID,
		OrigenCode:       c.PostForm("origen"),
		DestinoCode:      c.PostForm("destino"),
		FechaSalida:      fechaSalida,
		FechaRetorno:     fechaRetorno,
		Motivo:           c.PostForm("motivo"),
		Estado:           "SOLICITADO",
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

	c.HTML(http.StatusOK, "solicitud/show.html", gin.H{
		"Title":        "Detalle Solicitud #" + id,
		"Solicitud":    solicitud,
		"User":         c.MustGet("User"),
		"Step1":        step1,
		"Step2":        step2,
		"Step3":        step3,
		"MermaidGraph": mermaidGraph,
	})
}

func (ctrl *SolicitudController) Approve(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.service.Approve(id); err != nil {
		fmt.Printf("Error approving solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}

func (ctrl *SolicitudController) Reject(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.service.Reject(id); err != nil {
		fmt.Printf("Error rejecting solicitud: %v\n", err)
	}
	c.Redirect(http.StatusFound, "/solicitudes/"+id)
}
