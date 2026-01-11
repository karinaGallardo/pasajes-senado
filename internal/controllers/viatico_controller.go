package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type ViaticoController struct {
	viaticoService *services.ViaticoService
	solService     *services.SolicitudService
}

func NewViaticoController() *ViaticoController {
	return &ViaticoController{
		viaticoService: services.NewViaticoService(),
		solService:     services.NewSolicitudService(),
	}
}

func (ctrl *ViaticoController) Create(c *gin.Context) {
	id := c.Param("id")
	solicitud, err := ctrl.solService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Solicitud no encontrada")
		return
	}

	dias := 0.0
	if solicitud.FechaVuelta != nil && solicitud.FechaIda != nil {
		diff := solicitud.FechaVuelta.Sub(*solicitud.FechaIda)
		dias = diff.Hours() / 24
		if dias < 1 {
			dias = 1
		}
	} else {
		dias = 1
	}

	categorias, _ := ctrl.viaticoService.GetCategorias(c.Request.Context())

	utils.Render(c, "viatico/create", gin.H{
		"Title":       "Asignación de Viáticos",
		"Solicitud":   solicitud,
		"DefaultDias": fmt.Sprintf("%.1f", dias),
		"Categorias":  categorias,
	})
}

func (ctrl *ViaticoController) Calculate(c *gin.Context) {
}

func (ctrl *ViaticoController) Store(c *gin.Context) {
	solicitudID := c.Param("id")
	currentUser := appcontext.CurrentUser(c)
	if currentUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var req dtos.CreateViaticoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos: "+err.Error())
		return
	}

	dias, _ := strconv.ParseFloat(req.Dias, 64)
	montoDia, _ := strconv.ParseFloat(req.MontoDia, 64)
	porcentaje, _ := strconv.Atoi(req.Porcentaje)
	gastosRep := req.GastosRep == "on"

	layout := "2006-01-02"
	fechaDesde, _ := time.Parse(layout, req.FechaDesde)
	fechaHasta, _ := time.Parse(layout, req.FechaHasta)

	detalle := services.DetalleViaticoInput{
		FechaDesde: fechaDesde,
		FechaHasta: fechaHasta,
		Dias:       dias,
		Lugar:      req.Lugar,
		MontoDia:   montoDia,
		Porcentaje: porcentaje,
	}

	_, err := ctrl.viaticoService.RegistrarViatico(c.Request.Context(), solicitudID, []services.DetalleViaticoInput{detalle}, gastosRep, currentUser.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error asignando viático: "+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/solicitudes/"+solicitudID)
}

func (ctrl *ViaticoController) Print(c *gin.Context) {
	id := c.Param("id")
	viatico, err := ctrl.viaticoService.FindByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error retrieving viatico: "+err.Error())
		return
	}

	if c.GetHeader("HX-Request") == "true" {
		utils.Render(c, "viatico/modal_print", gin.H{
			"Viatico": viatico,
		})
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
	pdf.CellFormat(50, 6, "FORM-V-01", "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader, yHeader+16)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(50, 6, viatico.Codigo, "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+50, yHeader+5)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(100, 10, tr("FORMULARIO DE VIÁTICOS"), "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+50, yHeader+15)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(100, 5, tr("CÁMARA DE SENADORES"), "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+155, yHeader+2, 25, 0, false, "", 0, "")

	pdf.SetY(50)
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

	user := viatico.Usuario
	drawLabelBox("A FAVOR DE :", user.GetNombreCompleto(), 40, 150, false)
	drawLabelBox("C.I. :", user.CI, 40, 60, true)
	drawLabelBox("CARGO/ROL :", user.Rol.Nombre, 30, 60, false)
	drawLabelBox("UNIDAD :", "Senado Plurinacional", 40, 150, false)

	pdf.Ln(2)

	if viatico.Solicitud != nil {
		sol := viatico.Solicitud

		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(0, 8, tr("DATOS DE LA COMISIÓN"), "B", 1, "L", false, 0, "")
		pdf.Ln(2)

		drawLabelBox("SOLICITUD :", sol.Codigo, 40, 60, true)
		drawLabelBox("FECHA VIAJE :", sol.FechaIda.Format("02/01/2006"), 30, 60, false)
		drawLabelBox("MOTIVO :", sol.Motivo, 40, 150, false)
		drawLabelBox("LUGAR :", fmt.Sprintf("%s - %s", sol.Origen.Ciudad, sol.Destino.Ciudad), 40, 150, false)
	}

	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("DETALLE DE VIÁTICOS ASIGNADOS"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(30, 6, "Desde", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, "Hasta", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6, tr("Días"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, "Lugar", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 6, "Haber/Día (Bs)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 6, "%", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, "SubTotal (Bs)", "1", 0, "C", true, 0, "")
	pdf.Ln(6)

	pdf.SetFont("Arial", "", 8)
	for _, d := range viatico.Detalles {
		pdf.CellFormat(30, 6, d.FechaDesde.Format("02/01/2006"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, d.FechaHasta.Format("02/01/2006"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%.1f", d.Dias), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, tr(d.Lugar), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", d.MontoDia), "1", 0, "R", false, 0, "")
		pdf.CellFormat(15, 6, fmt.Sprintf("%d %%", d.Porcentaje), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f", d.SubTotal), "1", 0, "R", false, 0, "")
		pdf.Ln(6)
	}

	pdf.Ln(2)

	xTotals := 130.0
	wLabel := 40.0
	wValue := 30.0

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wLabel, 6, "TOTAL VIATICO :", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(wValue, 6, fmt.Sprintf("%.2f", viatico.MontoTotal), "1", 1, "R", false, 0, "")

	if viatico.TieneGastosRep {
		pdf.SetX(xTotals)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(wLabel, 6, "GASTOS REP. :", "1", 0, "R", true, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(wValue, 6, fmt.Sprintf("%.2f", viatico.MontoGastosRep), "1", 1, "R", false, 0, "")
	}

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wLabel, 6, "RETENCION (13%) :", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "", 9)
	totalRet := viatico.MontoRC_IVA + viatico.MontoRetencionGastos
	pdf.CellFormat(wValue, 6, fmt.Sprintf("%.2f", totalRet), "1", 1, "R", false, 0, "")

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(wLabel, 8, "LIQUIDO PAGABLE :", "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 10)
	totalLiq := viatico.MontoLiquido + viatico.MontoLiquidoGastos
	pdf.CellFormat(wValue, 8, fmt.Sprintf("%.2f", totalLiq), "1", 1, "R", false, 0, "")

	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Arial", "I", 9)
	literal := utils.NumeroALetras(totalLiq)
	pdf.CellFormat(190, 6, "Son: "+tr(literal)+" Bolivianos", "", 1, "L", false, 0, "")

	pdf.Ln(8)
	pdf.SetFont("Arial", "", 6)
	pdf.MultiCell(190, 3, tr("(CALCULO DE VIATICOS) Art.13, parr. III, inc. 1 : Cuando el hospedaje sea cubierto por algun organismo financiador u otra entidad publica organizadora, las Senadoras, Senadores o Servidores Públicos, declarados en comisión al interior como exterior del país, percibirán solamente el 70% de los viaticos"), "", "L", false)
	pdf.Ln(1)
	pdf.MultiCell(190, 3, tr("(CALCULO DE VIATICOS) Art.13, parr. III, inc. 2 : Cuando el hospedaje y alimentación, sean cubiertos por algun organismo financiador u otra entidad publica organizadora, las Senadoras, Senadores o Servidores Públicos, declarados en comisión al interior como al exterior del país, percibirán solamente el 25% de los viaticos"), "", "L", false)

	pdf.SetY(260)

	ySig := pdf.GetY()

	pdf.SetXY(20, ySig)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(20, ySig+2)
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(50, 4, tr("RECIBÍ CONFORME"), "", 1, "C", false, 0, "")
	pdf.SetXY(20, ySig+6)
	pdf.CellFormat(50, 4, tr(user.GetNombreCompleto()), "", 1, "C", false, 0, "")

	pdf.SetXY(80, ySig)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(80, ySig+2)
	pdf.CellFormat(50, 4, tr("ELABORADO POR"), "", 1, "C", false, 0, "")
	pdf.SetXY(80, ySig+6)
	currentUser := appcontext.CurrentUser(c)
	currentUserName := ""
	if currentUser != nil {
		currentUserName = currentUser.GetNombreCompleto()
	}
	pdf.CellFormat(50, 4, tr(currentUserName), "", 1, "C", false, 0, "")

	pdf.SetXY(140, ySig)
	pdf.Cell(50, 0, "__________________________")
	pdf.SetXY(140, ySig+2)
	pdf.CellFormat(50, 4, tr("AUTORIZADO"), "", 1, "C", false, 0, "")

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=VIATICO-%s.pdf", viatico.Codigo))
	pdf.Output(c.Writer)
}
