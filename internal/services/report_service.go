package services

import (
	"context"
	"fmt"
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

			aerolineaNombre := item.AerolineaSugerida
			if aerolinea, err := s.aerolineaRepo.FindByID(item.AerolineaSugerida); err == nil {
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

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(35, 230)
	pdf.Cell(60, 0, "__________________________")
	pdf.SetXY(35, 235)
	pdf.CellFormat(60, 4, tr("ENCARGADO DE PASAJES"), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "", 7)
	encargadoName := ""
	if solicitud.Usuario.Encargado != nil {
		encargadoName = solicitud.Usuario.Encargado.GetNombreCompleto()
	}
	pdf.CellFormat(60, 4, tr(encargadoName), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(115, 230)
	pdf.Cell(60, 0, "__________________________")
	pdf.SetXY(115, 235)
	pdf.CellFormat(60, 4, tr("SENADORA / SENADOR"), "", 1, "C", false, 0, "")
	pdf.SetX(115)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")

	pdf.SetY(250)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Generado electrónicamente por Sistema Pasajes Senado - %s", time.Now().Format("02/01/2006 15:04:05"))), "", 1, "C", false, 0, "")

	return pdf
}
