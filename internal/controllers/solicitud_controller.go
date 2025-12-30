package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type SolicitudController struct {
	service         *services.SolicitudService
	ciudadService   *services.CiudadService
	catalogoService *services.CatalogoService
	cupoService     *services.CupoService
	userService     *services.UsuarioService
	peopleService   *services.PeopleService
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:         services.NewSolicitudService(),
		ciudadService:   services.NewCiudadService(),
		catalogoService: services.NewCatalogoService(),
		cupoService:     services.NewCupoService(),
		userService:     services.NewUsuarioService(),
		peopleService:   services.NewPeopleService(),
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

	pdf := gofpdf.New("P", "mm", "A4", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	wHeader, hHeader := 190.0, 30.0

	pdf.SetLineWidth(0.5)
	pdf.Rect(xHeader, yHeader, wHeader, hHeader, "D")

	pdf.Line(xHeader+50, yHeader, xHeader+50, yHeader+hHeader)
	pdf.Line(xHeader+150, yHeader, xHeader+150, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+6)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(50, 6, "FORM-PV-01", "", 1, "C", false, 0, "")

	displayCode := solicitud.Codigo
	if displayCode == "" {
		displayCode = solicitud.ID
		if len(displayCode) > 8 {
			displayCode = displayCode[:8]
		}
	}

	pdf.SetXY(xHeader, yHeader+16)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(50, 6, "SOL-"+displayCode, "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+50, yHeader+5)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(100, 10, "FORMULARIO DE SOLICITUD", "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+50, yHeader+15)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(100, 5, "PASAJES AEREOS PARA SENADORAS Y", "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+50, yHeader+20)
	pdf.CellFormat(100, 5, "SENADORES", "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+155, yHeader+2, 25, 0, false, "", 0, "")

	pdf.SetY(50)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 5, fmt.Sprintf("Fecha de Solicitud: %s", solicitud.CreatedAt.Format("02/01/2006 15:04")), "", 1, "C", false, 0, "")

	drawLabelBox := func(label, value string, wLabel, wBox float64, sameLine bool) {
		h := 6.0
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(wLabel, h, tr(label), "", 0, "R", false, 0, "")

		pdf.SetFont("Arial", "", 9)
		if len(value) > 75 {
			value = value[:72] + "..."
		}
		pdf.CellFormat(wBox, h, "  "+tr(value), "1", 0, "L", false, 0, "")

		if !sameLine {
			pdf.Ln(h + 2)
		}
	}

	drawLabelBox("NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 40, 150, false)
	drawLabelBox("C.I. :", solicitud.Usuario.CI, 40, 60, true)
	drawLabelBox("TEL. REF :", solicitud.Usuario.Phone, 30, 60, false)

	origenUser := ""
	if solicitud.Usuario.Origen != nil {
		origenUser = solicitud.Usuario.Origen.Nombre
	}

	personaView, errMongo := ctrl.peopleService.FindSenatorDataByCI(solicitud.Usuario.CI)

	tipoUsuario := solicitud.Usuario.Tipo
	unit := "COMISION"

	if errMongo == nil && personaView != nil {
		senadorData := personaView.SenadorData
		if senadorData.Departamento != "" {
			origenUser = fmt.Sprintf("%s (%s)", senadorData.Departamento, senadorData.Sigla)
			if senadorData.Gestion != "" {
				origenUser += fmt.Sprintf(" | %s", senadorData.Gestion)
			}
		}
		if senadorData.Tipo != "" {
			tipoUsuario = senadorData.Tipo
		}

		if personaView.FuncionarioPermanente.ItemData.Unit != "" {
			unit = personaView.FuncionarioPermanente.ItemData.Unit
		} else if personaView.FuncionarioEventual.UnitData.Name != "" {
			unit = personaView.FuncionarioEventual.UnitData.Name
		}
	}

	drawLabelBox("SENADOR POR EL DPTO. :", origenUser, 40, 60, true)

	isTitular := strings.Contains(strings.ToUpper(tipoUsuario), "TITULAR")
	isSuplente := strings.Contains(strings.ToUpper(tipoUsuario), "SUPLENTE")

	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(25, 6, "TITULAR", "", 0, "R", false, 0, "")

	xCheck, yCheck := pdf.GetX(), pdf.GetY()
	pdf.Rect(xCheck+1, yCheck+1, 4, 4, "D")
	if isTitular {
		pdf.Text(xCheck+1.5, yCheck+4.5, "X")
	}
	pdf.SetX(xCheck + 10)

	pdf.CellFormat(20, 6, "SUPLENTE", "", 0, "R", false, 0, "")
	xCheck, yCheck = pdf.GetX(), pdf.GetY()
	pdf.Rect(xCheck+1, yCheck+1, 4, 4, "D")
	if isSuplente {
		pdf.Text(xCheck+1.5, yCheck+4.5, "X")
	}
	pdf.Ln(8)

	drawLabelBox("UNIDAD FUNCIONAL :", unit, 40, 150, false)

	fechaSol := solicitud.CreatedAt.Format("02/01/2006")
	horaSol := solicitud.CreatedAt.Format("15:04")
	drawLabelBox("FECHA DE SOLICITUD :", fechaSol, 40, 60, true)
	drawLabelBox("HORA :", horaSol, 30, 60, false)

	concepto := ""
	if solicitud.TipoSolicitud != nil {
		concepto = solicitud.TipoSolicitud.Nombre
	}
	pdf.SetFont("Arial", "B", 7)
	pdf.SetXY(110, pdf.GetY()+5)
	pdf.Cell(0, 5, tr("Si, el concepto es POR DERECHO :"))
	pdf.SetXY(10, pdf.GetY()+5)

	drawLabelBox("CONCEPTO DE VIAJE :", concepto, 40, 60, true)

	mesYNum := ""
	if strings.Contains(strings.ToUpper(concepto), "DERECHO") {
		mesES := map[string]string{"January": "ENERO", "February": "FEBRERO", "March": "MARZO", "April": "ABRIL", "May": "MAYO", "June": "JUNIO", "July": "JULIO", "August": "AGOSTO", "September": "SEPTIEMBRE", "October": "OCTUBRE", "November": "NOVIEMBRE", "December": "DICIEMBRE"}
		mesYNum = mesES[solicitud.FechaSalida.Month().String()]
	}
	drawLabelBox("MES Y N° DE PASAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	tipoItinerario := "IDA"
	routeText := fmt.Sprintf("%s - %s", solicitud.Origen.Nombre, solicitud.Destino.Nombre)
	if solicitud.TipoItinerario != nil {
		if strings.Contains(strings.ToUpper(solicitud.TipoItinerario.Nombre), "VUELTA") {
			tipoItinerario = "IDA Y VUELTA"
			routeText += fmt.Sprintf(" - %s", solicitud.Origen.Nombre)
		}
	}

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr(fmt.Sprintf("SOLICITA PASAJES DE %s EN LA SIGUIENTE RUTA", tipoItinerario)), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 8, tr(routeText), "1", 1, "C", false, 0, "")
	pdf.Ln(8)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 8, tr("JUSTIFICACION / MOTIVO"), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(190, 6, tr(solicitud.Motivo), "1", "L", false)

	pdf.SetY(230)

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(20, 230)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(20, 235)
	pdf.CellFormat(50, 4, "SOLICITANTE", "", 1, "C", false, 0, "")
	pdf.SetX(20)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(50, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(80, 230)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(80, 235)
	pdf.CellFormat(50, 4, tr("AUTORIZACIÓN"), "", 1, "C", false, 0, "")
	pdf.SetX(80)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(50, 4, tr("Jefe Inmediato / Autoridad"), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(140, 230)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(140, 235)
	pdf.CellFormat(50, 4, tr("ADMINISTRACIÓN"), "", 1, "C", false, 0, "")
	pdf.SetX(140)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(50, 4, tr("Verificación Cupo/Ppto"), "", 1, "C", false, 0, "")

	pdf.SetY(270)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Generado electrónicamente por Sistema Pasajes Senado - %s", time.Now().Format("02/01/2006 15:04:05"))), "", 1, "C", false, 0, "")
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=FORM-PV01-%s.pdf", solicitud.ID))
	pdf.Output(c.Writer)
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
