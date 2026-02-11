package services

import (
	"bytes"
	"context"
	"fmt"

	"os"
	"os/exec"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

type ReportService struct {
	solicitudRepo *repositories.SolicitudRepository
	aerolineaRepo *repositories.AerolineaRepository
}

func NewReportService() *ReportService {
	return &ReportService{
		solicitudRepo: repositories.NewSolicitudRepository(),
		aerolineaRepo: repositories.NewAerolineaRepository(),
	}
}

func (s *ReportService) GeneratePV01(ctx context.Context, solicitud *models.Solicitud, personaView *models.MongoPersonaView, mode string) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	hHeader := 22.0

	pdf.SetLineWidth(0.1)
	pdf.Line(xHeader, yHeader+hHeader, xHeader+167, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+3)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-01", "", 1, "C", false, 0, "")

	pdf.SetX(xHeader)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(40, 8, fmt.Sprintf("%s", solicitud.Codigo), "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE SOLICITUD"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES AEREOS PARA SENADORAS Y SENADORES"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+16)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(115, 6, fmt.Sprintf("GESTION: %s", solicitud.CreatedAt.Format("2006")), "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+160, yHeader-6, 30, 0, false, "", 0, "")

	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)

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

	// Body
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
	}

	if solicitud.Usuario.Oficina != nil {
		unit = solicitud.Usuario.Oficina.Detalle
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
	// Note: simplified logic compared to CupoReport which checks ConceptoViaje
	if solicitud.TipoSolicitud != nil && solicitud.TipoSolicitud.ConceptoViaje != nil {
		concepto = solicitud.TipoSolicitud.ConceptoViaje.Nombre
	}

	pdf.SetXY(10, pdf.GetY()+5)
	drawLabelBox("CONCEPTO DE VIAJE :", concepto, 40, 60, true)

	mesYNum := ""
	if strings.Contains(strings.ToUpper(concepto), "DERECHO") {
		var dateRef *time.Time
		for _, item := range solicitud.Items {
			if item.Fecha != nil {
				dateRef = item.Fecha
				break
			}
		}
		if dateRef != nil {
			mesYNum = utils.TranslateMonth(dateRef.Month())
		}
	}
	drawLabelBox("MES DE VIAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	// SEGMENTS LOGIC
	drawSegment := func(title string, item *models.SolicitudItem) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+title), "B", 1, "L", false, 0, "")

		if item != nil {
			fecha := "-"
			hora := "-"
			if item.Fecha != nil {
				fecha = utils.FormatDateShortES(*item.Fecha)
				if len(fecha) > 5 {
					fecha = fecha[:len(fecha)-5] // Remove year
				}
				hora = item.Fecha.Format("15:04")
			} else if item.Hora != "" {
				hora = item.Hora
			}
			fechaSol := solicitud.CreatedAt.Format("02/01/2006")
			horaSol := solicitud.CreatedAt.Format("15:04")

			origenStr := item.OrigenIATA
			if item.Origen != nil {
				origenStr = item.Origen.Ciudad
			}
			destStr := item.DestinoIATA
			if item.Destino != nil {
				destStr = item.Destino.Ciudad
			}

			rut := fmt.Sprintf("%s  >>  %s", origenStr, destStr)

			// Table Header
			pdf.SetFont("Arial", "B", 7)
			pdf.SetFillColor(245, 245, 245)

			// Headers
			pdf.CellFormat(32, 8, tr("FECHA/HORA SOLICITUD"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(35, 8, tr("AEROLÍNEA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(52, 8, tr("RUTA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(30, 8, tr("FECHA VIAJE"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(18, 8, tr("HORA VIAJE"), "1", 1, "C", true, 0, "")

			// Data Row
			pdf.SetFont("Arial", "", 9)

			pdf.CellFormat(32, 8, fmt.Sprintf("%s %s", fechaSol, horaSol), "1", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 128)
			pdf.CellFormat(25, 8, tr(item.GetEstado()), "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 7)

			aerolineaNombre := solicitud.AerolineaSugerida
			if aerolinea, err := s.aerolineaRepo.FindByID(solicitud.AerolineaSugerida); err == nil {
				if aerolinea.Sigla != "" {
					aerolineaNombre = aerolinea.Sigla
				} else {
					aerolineaNombre = aerolinea.Nombre
				}
			}

			pdf.CellFormat(35, 8, tr(aerolineaNombre), "1", 0, "C", false, 0, "")
			pdf.CellFormat(52, 8, tr(rut), "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(30, 8, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 8, hora, "1", 1, "C", false, 0, "")

		} else {
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(190, 10, tr(" TRAMO NO SOLICITADO / DISPONIBLE"), "", 1, "C", false, 0, "")
		}
		pdf.Ln(2)
	}

	var idaItem, vueltaItem *models.SolicitudItem
	for i := range solicitud.Items {
		item := &solicitud.Items[i]
		switch item.Tipo {
		case models.TipoSolicitudItemIda:
			idaItem = item
		case models.TipoSolicitudItemVuelta:
			vueltaItem = item
		}
	}

	switch mode {
	case "ida":
		vueltaItem = nil
	case "vuelta":
		idaItem = nil
	}

	pdf.Ln(2)
	drawSegment("TRAYECTO DE IDA", idaItem)
	pdf.Ln(4)
	drawSegment("TRAYECTO DE VUELTA", vueltaItem)

	pdf.Ln(8)

	authVal := solicitud.Autorizacion
	if authVal == "" {
		authVal = "PD"
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 6, tr("AUTORIZACIÓN :"), "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(150, 6, "  "+tr(authVal), "1", 1, "L", false, 0, "")

	pdf.Ln(2)

	// Motivo
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(190, 6, tr("JUSTIFICACIÓN / MOTIVO"), "L,R,T", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.MultiCell(190, 6, tr(solicitud.Motivo), "L,R,B", "L", false)

	pdf.Ln(5)

	pdf.SetY(220)

	// Signatures
	pdf.SetY(220)
	pdf.SetFont("Arial", "B", 7)

	// Left (Encargado)
	pdf.Line(35, 230, 95, 230)
	pdf.SetXY(35, 232)
	pdf.CellFormat(60, 4, tr("ENCARGADO DE PASAJES"), "", 1, "C", false, 0, "")
	if solicitud.Usuario.Encargado != nil {
		pdf.SetX(35)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(60, 4, tr(solicitud.Usuario.Encargado.GetNombreCompleto()), "", 1, "C", false, 0, "")
	}

	// Right (Senador)
	pdf.Line(115, 230, 175, 230)
	pdf.SetXY(115, 232)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr("SENADORA / SENADOR"), "", 1, "C", false, 0, "")
	pdf.SetX(115)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")
	pdf.SetX(115)
	pdf.SetFont("Arial", "I", 6)
	pdf.CellFormat(60, 3, tr("(FIRMA Y SELLO)"), "", 1, "C", false, 0, "")

	pdf.SetY(250)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Generado electrónicamente por Sistema Pasajes Senado - %s", time.Now().Format("02/01/2006 15:04:05"))), "", 1, "C", false, 0, "")

	return pdf
}

func (s *ReportService) GeneratePV05(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	solicitud := descargo.Solicitud
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	hHeader := 22.0

	pdf.SetLineWidth(0.1)
	pdf.Line(xHeader, yHeader+hHeader, xHeader+167, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+3)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-05", "", 1, "C", false, 0, "")

	pdf.SetX(xHeader)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(40, 8, fmt.Sprintf("%s", solicitud.Codigo), "", 1, "C", false, 0, "") // Reuse code or generate new one? usually keeps same tracking
	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE DESCARGO"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES AEREOS PARA SENADORAS Y SENADORES"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+16)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(115, 6, fmt.Sprintf("GESTION: %s", solicitud.CreatedAt.Format("2006")), "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+160, yHeader-6, 30, 0, false, "", 0, "")

	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)

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

	// Body
	drawLabelBox("NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 40, 150, false)
	drawLabelBox("C.I. :", solicitud.Usuario.CI, 40, 60, true)
	drawLabelBox("TEL. REF :", solicitud.Usuario.Phone, 30, 60, false)

	origenUser := ""
	if solicitud.Usuario.Origen != nil {
		origenUser = solicitud.Usuario.Origen.Ciudad
	}

	tipoUsuario := solicitud.Usuario.Tipo
	// unit := "COMISION"

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

	// Month of travel
	mes := ""
	var mainDate *time.Time
	for _, it := range solicitud.Items {
		if it.Fecha != nil {
			mainDate = it.Fecha
			break
		}
	}
	if mainDate != nil {
		mes = utils.TranslateMonth(mainDate.Month())
	}
	drawLabelBox("CORRESPONDIENTE AL MES DE :", mes, 50, 60, false)
	pdf.Ln(2)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 8, tr("DESCARGO DE PASAJES (ADJUNTAR PASES A BORDO)"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	// Function to draw a table for a set of rows
	drawSubTable := func(subTitle string, headerBoleto string, rows []models.DetalleItinerarioDescargo) {
		if subTitle != "" {
			pdf.SetFillColor(240, 240, 240)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 5, tr(subTitle), "1", 1, "C", true, 0, "")
		}

		// Headers
		pdf.SetFillColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(70, 5, tr("RUTA"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 5, tr("FECHA DE VIAJE"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 5, tr(headerBoleto), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 5, tr("N° PASE A BORDO"), "1", 1, "C", false, 0, "")

		// Data Rows
		if len(rows) == 0 {
			// At least one empty row to match visual
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(35, 6, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 6, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 6, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(50, 6, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 6, "", "1", 1, "C", false, 0, "")
			return
		}

		pdf.SetFont("Arial", "", 8)
		for _, r := range rows {
			orig, dest := "", ""
			parts := strings.Split(r.Ruta, "-")
			if len(parts) >= 2 {
				orig = strings.TrimSpace(parts[0])
				dest = strings.TrimSpace(parts[1])
			} else {
				orig = r.Ruta
			}

			pdf.CellFormat(35, 6, tr(orig), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 6, tr(dest), "1", 0, "C", false, 0, "")

			fecha := ""
			if r.Fecha != nil {
				fecha = r.Fecha.Format("02/01/2006")
			}
			pdf.CellFormat(30, 6, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(50, 6, tr(r.Boleto), "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 6, tr(r.NumeroPaseAbordo), "1", 1, "C", false, 0, "")
		}
	}

	drawSegmentBlock := func(title string, typeOrig, typeRepro models.TipoDetalleItinerario) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+title), "", 1, "L", false, 0, "")
		pdf.Ln(1)

		var origRows, reproRows []models.DetalleItinerarioDescargo
		for _, d := range descargo.DetallesItinerario {
			// Skip returns in main table
			if d.EsDevolucion {
				continue
			}
			switch d.Tipo {
			case typeOrig:
				origRows = append(origRows, d)
			case typeRepro:
				reproRows = append(reproRows, d)
			}
		}

		// Table 1: Original
		drawSubTable("", "N° BOLETO ORIGINAL", origRows)

		// Table 2: Reprogramación
		if len(reproRows) > 0 {
			pdf.Ln(1)
			drawSubTable("REPROGRAMACIÓN", "N° BOLETO REPROGRAMADO", reproRows)
		}
	}

	drawSegmentBlock("TRAMO DE IDA", models.TipoDetalleIdaOriginal, models.TipoDetalleIdaReprogramada)
	pdf.Ln(4)
	drawSegmentBlock("TRAMO DE RETORNO", models.TipoDetalleVueltaOriginal, models.TipoDetalleVueltaReprogramada)
	pdf.Ln(6)

	// Devolucion
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" DEVOLUCIÓN DE PASAJES POR DERECHO"), "B", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(190, 5, tr(" (En caso de no haber utilizado el boleto emitido o un tramo informar en el siguiente cuadro)"), "", 1, "L", false, 0, "")

	// Collect Returns grouped
	var returnsIda, returnsVuelta []models.DetalleItinerarioDescargo
	for _, d := range descargo.DetallesItinerario {
		if d.EsDevolucion {
			if strings.Contains(string(d.Tipo), "IDA") {
				returnsIda = append(returnsIda, d)
			} else if strings.Contains(string(d.Tipo), "VUELTA") {
				returnsVuelta = append(returnsVuelta, d)
			}
		}
	}

	drawReturnTable := func(subTitle string, rows []models.DetalleItinerarioDescargo) {
		if subTitle != "" {
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 6, tr(subTitle), "", 1, "L", false, 0, "")
		}

		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(120, 6, tr("RUTA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(70, 6, tr("N° BOLETO"), "1", 1, "C", true, 0, "")

		if len(rows) > 0 {
			pdf.SetFont("Arial", "", 8)
			for _, r := range rows {
				orig, dest := "", ""
				parts := strings.Split(r.Ruta, "-")
				if len(parts) >= 2 {
					orig = strings.TrimSpace(parts[0])
					dest = strings.TrimSpace(parts[1])
				} else {
					orig = r.Ruta
				}

				pdf.CellFormat(60, 6, tr(orig), "1", 0, "C", false, 0, "")
				pdf.CellFormat(60, 6, tr(dest), "1", 0, "C", false, 0, "")
				pdf.CellFormat(70, 6, tr(r.Boleto), "1", 1, "C", false, 0, "")
			}
		} else {
			pdf.CellFormat(60, 8, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(60, 8, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(70, 8, "", "1", 1, "C", false, 0, "")
		}
	}

	if len(returnsIda) > 0 || len(returnsVuelta) > 0 {
		if len(returnsIda) > 0 {
			drawReturnTable("TRAMO DE IDA", returnsIda)
		}
		if len(returnsVuelta) > 0 {
			if len(returnsIda) > 0 {
				pdf.Ln(2)
			}
			drawReturnTable("TRAMO DE RETORNO", returnsVuelta)
		}
	} else {
		drawReturnTable("", nil)
	}
	pdf.Ln(10)

	// Signatures
	pdf.SetY(220)
	pdf.SetFont("Arial", "B", 8)

	// Left (Senador)
	pdf.Line(35, 230, 95, 230) // Line width 60 (35 to 95)
	pdf.SetXY(35, 232)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr("SENADORA / SENADOR"), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "I", 6)
	pdf.CellFormat(60, 3, tr("(FIRMA Y SELLO)"), "", 1, "C", false, 0, "")

	// Right (Responsable Presentacion)
	pdf.Line(110, 230, 180, 230) // Line width 70 (110 to 180)
	pdf.SetXY(110, 232)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(70, 4, tr("RESPONSABLE PRESENTACION DEL DESCARGO"), "", 1, "C", false, 0, "")
	pdf.SetX(110)

	pdf.SetY(250)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Generado electrónicamente por Sistema Pasajes Senado - %s", time.Now().Format("02/01/2006 15:04:05"))), "", 1, "C", false, 0, "")

	return pdf
}

func (s *ReportService) GeneratePV05Complete(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) ([]byte, error) {
	// 1. Generar el PDF Base PV-05
	pdf := s.GeneratePV05(ctx, descargo, personaView)

	// Crear archivos temporales para la unión
	tmpBase, err := os.CreateTemp("", "pv05_base_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBase.Name())

	if err := pdf.OutputFileAndClose(tmpBase.Name()); err != nil {
		return nil, err
	}

	// 2. Recolectar rutas de archivos que existan (Billetes Electrónicos + Pases a Bordo)
	var filesToMerge []string
	filesToMerge = append(filesToMerge, tmpBase.Name())

	// 2.1 Billetes Electrónicos (Pasajes emitidos)
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, pasaje := range item.Pasajes {
				if pasaje.Archivo != "" {
					if _, err := os.Stat(pasaje.Archivo); err == nil {
						filesToMerge = append(filesToMerge, pasaje.Archivo)
					}
				}
			}
		}
	}

	// 2.2 Pases a Bordo (Cargados en el descargo)
	for _, det := range descargo.DetallesItinerario {
		if det.ArchivoPaseAbordo != "" {
			if _, err := os.Stat(det.ArchivoPaseAbordo); err == nil {
				filesToMerge = append(filesToMerge, det.ArchivoPaseAbordo)
			}
		}
	}

	// 3. Si solo hay un archivo (el base), retornarlo directamente
	if len(filesToMerge) == 1 {
		return os.ReadFile(tmpBase.Name())
	}

	// 4. Unir usando pdftk (que ya verificamos que existe)
	tmpFinal, err := os.CreateTemp("", "pv05_final_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFinal.Name())

	args := append(filesToMerge, "cat", "output", tmpFinal.Name())
	cmd := exec.Command("pdftk", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error al unir PDFs con pdftk: %v - %s", err, stderr.String())
	}

	return os.ReadFile(tmpFinal.Name())
}
