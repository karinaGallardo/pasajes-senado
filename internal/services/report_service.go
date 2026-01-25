package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ReportService struct {
}

func NewReportService() *ReportService {
	return &ReportService{}
}

func (s *ReportService) GeneratePV01(ctx context.Context, solicitud *models.Solicitud, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
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
		origenUser = solicitud.Usuario.Origen.Ciudad
	}

	tipoUsuario := solicitud.Usuario.Tipo
	unit := "COMISION"

	if personaView != nil {
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

		if dep := utils.GetString(personaView.Dependencia); dep != "" {
			unit = dep
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
		if solicitud.FechaIda != nil {
			mesYNum = utils.TranslateMonth(solicitud.FechaIda.Month())
		} else if solicitud.FechaVuelta != nil {
			mesYNum = utils.TranslateMonth(solicitud.FechaVuelta.Month())
		}
	}
	drawLabelBox("MES Y N° DE PASAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	tipoItinerario := "IDA"
	routeText := fmt.Sprintf("%s - %s", solicitud.Origen.Ciudad, solicitud.Destino.Ciudad)

	if solicitud.TipoItinerario != nil {
		switch solicitud.TipoItinerario.Codigo {
		case "IDA_VUELTA":
			tipoItinerario = "IDA Y VUELTA"
			routeText += fmt.Sprintf(" - %s", solicitud.Origen.Ciudad)
		case "SOLO_IDA":
			tipoItinerario = "IDA"
		default:
			if strings.Contains(strings.ToUpper(solicitud.TipoItinerario.Nombre), "VUELTA") && !strings.Contains(strings.ToUpper(solicitud.TipoItinerario.Codigo), "SOLO") {
				tipoItinerario = "IDA Y VUELTA"
				routeText += fmt.Sprintf(" - %s", solicitud.Origen.Ciudad)
			}
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

	return pdf
}
