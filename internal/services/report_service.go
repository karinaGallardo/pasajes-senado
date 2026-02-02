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

func (s *ReportService) GenerateCupoSolicitudesPDF(ctx context.Context, cupoItemID string) ([]byte, error) {
	solicitudes, err := s.solicitudRepo.WithContext(ctx).FindByCupoDerechoItemID(cupoItemID)
	if err != nil {
		return nil, err
	}
	hasIda := false
	hasVuelta := false

	for _, req := range solicitudes {
		if req.TipoItinerario != nil {
			if req.TipoItinerario.Codigo == "SOLO_IDA" || req.TipoItinerario.Codigo == "IDA_VUELTA" {
				hasIda = true
			}
			if req.TipoItinerario.Codigo == "SOLO_VUELTA" || req.TipoItinerario.Codigo == "IDA_VUELTA" {
				hasVuelta = true
			}
		}
	}

	if !hasIda && !hasVuelta {
		return nil, fmt.Errorf("se requiere al menos una solicitud (Ida o Vuelta) para generar este reporte")
	}

	pdf := s.GenerateCupoReport(ctx, solicitudes)

	var buf strings.Builder
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func (s *ReportService) GenerateCupoReport(ctx context.Context, solicitudes []models.Solicitud) *gofpdf.Fpdf {

	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	hHeader := 22.0
	// wHeader, hHeader := 190.0, 22.0

	pdf.SetLineWidth(0.1)
	// pdf.Rect(xHeader, yHeader, wHeader, hHeader, "D")
	// pdf.Line(xHeader+40, yHeader, xHeader+40, yHeader+hHeader)
	// pdf.Line(xHeader+155, yHeader, xHeader+155, yHeader+hHeader)
	pdf.Line(xHeader, yHeader+hHeader, xHeader+167, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+9)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-01", "", 1, "C", false, 0, "")

	mainSol := models.Solicitud{}
	if len(solicitudes) > 0 {
		mainSol = solicitudes[0]
	}

	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE SOLICITUD"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES AEREOS PARA SENADORAS Y SENADORES"), "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+160, yHeader-6, 30, 0, false, "", 0, "")

	pdf.SetY(40) // define el inicio del cuerpo del formulario
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

	drawLabelBox("NOMBRE Y APELLIDOS :", mainSol.Usuario.GetNombreCompleto(), 40, 150, false)
	drawLabelBox("C.I. :", mainSol.Usuario.CI, 40, 60, true)
	drawLabelBox("TEL. REF :", mainSol.Usuario.Phone, 30, 60, false)

	origenUser := ""
	if mainSol.Usuario.Departamento != nil {
		origenUser = mainSol.Usuario.Departamento.Nombre
	} else if mainSol.Usuario.Origen != nil {
		origenUser = mainSol.Usuario.Origen.Ciudad
	}
	unit := "COMISION"
	if mainSol.Usuario.Oficina != nil {
		unit = mainSol.Usuario.Oficina.Detalle
	}

	drawLabelBox("SENADOR POR EL DPTO. :", origenUser, 40, 60, true)

	tipoUsuario := mainSol.Usuario.Tipo
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

	fechaSol := mainSol.CreatedAt.Format("02/01/2006")
	horaSol := mainSol.CreatedAt.Format("15:04")
	drawLabelBox("FECHA DE SOLICITUD :", fechaSol, 40, 60, true)
	drawLabelBox("HORA :", horaSol, 30, 60, false)

	concepto := ""
	if mainSol.TipoSolicitud != nil && mainSol.TipoSolicitud.ConceptoViaje != nil {
		concepto = mainSol.TipoSolicitud.ConceptoViaje.Nombre
	}
	pdf.SetXY(10, pdf.GetY()+5)

	drawLabelBox("CONCEPTO DE VIAJE :", concepto, 40, 60, true)

	mesYNum := ""
	if mainSol.CupoDerechoItem != nil {
		monthName := utils.GetMonthName(mainSol.CupoDerechoItem.Mes)
		mesYNum = fmt.Sprintf("%s / %s", strings.ToUpper(monthName), mainSol.CupoDerechoItem.Semana)
	} else if strings.Contains(strings.ToUpper(concepto), "DERECHO") {
		for _, s := range solicitudes {
			if s.FechaIda != nil {
				mesYNum = utils.TranslateMonth(s.FechaIda.Month())
				break
			}
		}
	}
	drawLabelBox("MES Y N° DE CUPO :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	var ida, vuelta *models.Solicitud
	for i := range solicitudes {
		s := &solicitudes[i]
		if s.TipoItinerario != nil {
			codigo := s.TipoItinerario.Codigo
			switch codigo {
			case "SOLO_IDA":
				ida = s
			case "SOLO_VUELTA":
				vuelta = s
			default:
				if ida == nil {
					ida = s
				}
			}
		}
	}

	drawSegment := func(title string, sol *models.Solicitud) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+title), "B", 1, "L", false, 0, "")

		if sol != nil {
			fecha := "-"
			hora := "-"
			if sol.FechaIda != nil {
				// fecha = sol.FechaIda.Format("02/01/2006")
				fecha = utils.FormatDateShortES(*sol.FechaIda)
				hora = sol.FechaIda.Format("15:04")
			}
			fechaSol := sol.CreatedAt.Format("02/01/2006")
			horaSol := sol.CreatedAt.Format("15:04")

			rut := fmt.Sprintf("%s  >>  %s", sol.Origen.Ciudad, sol.Destino.Ciudad)

			// Table Header
			pdf.SetFont("Arial", "B", 7)
			pdf.SetFillColor(245, 245, 245)

			// Headers
			pdf.CellFormat(30, 8, tr("CÓDIGO SOLICITUD"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(30, 8, tr("FECHA/HORA SOLICITUD"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(20, 8, tr("AEROLÍNEA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 8, tr("RUTA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, tr("FECHA VIAJE"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(20, 8, tr("HORA VIAJE"), "1", 1, "C", true, 0, "")

			// Data Row
			pdf.SetFont("Arial", "", 7)
			pdf.CellFormat(30, 6, sol.Codigo, "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("%s %s", fechaSol, horaSol), "1", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 128)
			pdf.CellFormat(25, 6, tr(sol.GetEstado()), "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 7)

			aerolineaNombre := sol.AerolineaSugerida
			if aerolinea, err := s.aerolineaRepo.FindByID(sol.AerolineaSugerida); err == nil {
				if aerolinea.Sigla != "" {
					aerolineaNombre = aerolinea.Sigla
				} else {
					aerolineaNombre = aerolinea.Nombre
				}
			}

			pdf.CellFormat(20, 6, tr(aerolineaNombre), "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 6, tr(rut), "1", 0, "C", false, 0, "")
			pdf.CellFormat(25, 6, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(20, 6, hora, "1", 1, "C", false, 0, "")

			// Motivo
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 6, tr("JUSTIFICACIÓN / MOTIVO"), "L,R", 1, "L", true, 0, "")
			pdf.SetFont("Arial", "", 8)
			pdf.MultiCell(190, 6, tr(sol.Motivo), "L,R,B", "L", false)

		} else {
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(190, 10, tr(" TRAMO NO SOLICITADO / DISPONIBLE"), "", 1, "C", false, 0, "")
		}
		pdf.Ln(2)
	}

	drawSegment("TRAYECTO DE IDA", ida)
	drawSegment("TRAYECTO DE VUELTA", vuelta)

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
	if mainSol.Usuario.Encargado != nil {
		encargadoName = mainSol.Usuario.Encargado.GetNombreCompleto()
	}
	pdf.CellFormat(60, 4, tr(encargadoName), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetXY(115, 230)
	pdf.Cell(60, 0, "__________________________")
	pdf.SetXY(115, 235)
	pdf.CellFormat(60, 4, tr("SENADORA / SENADOR"), "", 1, "C", false, 0, "")
	pdf.SetX(115)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(mainSol.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")

	pdf.SetY(250)
	pdf.SetFont("Arial", "I", 8)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Generado electrónicamente por Sistema Pasajes Senado - %s", time.Now().Format("02/01/2006 15:04:05"))), "", 1, "C", false, 0, "")

	return pdf
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

	pdf.SetXY(xHeader, yHeader+9)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-01", "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE SOLICITUD"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES AEREOS PARA SENADORAS Y SENADORES"), "", 1, "C", false, 0, "")

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
		dateRef := solicitud.FechaIda
		if dateRef == nil {
			dateRef = solicitud.FechaVuelta
		}
		if dateRef != nil {
			mesYNum = utils.TranslateMonth(dateRef.Month())
		}
	}
	drawLabelBox("MES DE VIAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	// SEGMENTS LOGIC
	drawSegment := func(title string, sol *models.Solicitud) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+title), "B", 1, "L", false, 0, "")

		if sol != nil {
			fecha := "-"
			hora := "-"
			if sol.FechaIda != nil {
				fecha = utils.FormatDateShortES(*sol.FechaIda)
				hora = sol.FechaIda.Format("15:04")
			}
			fechaSol := sol.CreatedAt.Format("02/01/2006")
			horaSol := sol.CreatedAt.Format("15:04")

			rut := fmt.Sprintf("%s  >>  %s", sol.Origen.Ciudad, sol.Destino.Ciudad)

			// Table Header
			pdf.SetFont("Arial", "B", 7)
			pdf.SetFillColor(245, 245, 245)

			// Headers
			pdf.CellFormat(30, 8, tr("CÓDIGO SOLICITUD"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(30, 8, tr("FECHA/HORA SOLICITUD"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(20, 8, tr("AEROLÍNEA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 8, tr("RUTA"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(25, 8, tr("FECHA VIAJE"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(20, 8, tr("HORA VIAJE"), "1", 1, "C", true, 0, "")

			// Data Row
			pdf.SetFont("Arial", "", 7)
			pdf.CellFormat(30, 6, sol.Codigo, "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("%s %s", fechaSol, horaSol), "1", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 128)
			pdf.CellFormat(25, 6, tr(sol.GetEstado()), "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 7)

			aerolineaNombre := sol.AerolineaSugerida
			if aerolinea, err := s.aerolineaRepo.FindByID(sol.AerolineaSugerida); err == nil {
				if aerolinea.Sigla != "" {
					aerolineaNombre = aerolinea.Sigla
				} else {
					aerolineaNombre = aerolinea.Nombre
				}
			}

			pdf.CellFormat(20, 6, tr(aerolineaNombre), "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 6, tr(rut), "1", 0, "C", false, 0, "")
			pdf.CellFormat(25, 6, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(20, 6, hora, "1", 1, "C", false, 0, "")

			// Motivo
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 6, tr("JUSTIFICACIÓN / MOTIVO"), "L,R", 1, "L", true, 0, "")
			pdf.SetFont("Arial", "", 8)
			pdf.MultiCell(190, 6, tr(sol.Motivo), "L,R,B", "L", false)

		} else {
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(190, 10, tr(" TRAMO NO SOLICITADO / DISPONIBLE"), "", 1, "C", false, 0, "")
		}
		pdf.Ln(2)
	}

	var ida, vuelta *models.Solicitud

	// Determine configuration based on Mode and Itinerary
	if mode == "ida" {
		tmp := *solicitud
		ida = &tmp
	} else if mode == "vuelta" {
		if solicitud.FechaVuelta != nil {
			tmp := *solicitud
			tmp.FechaIda = tmp.FechaVuelta
			tmp.Origen, tmp.Destino = tmp.Destino, tmp.Origen
			vuelta = &tmp
		}
	} else {
		// Auto / Complete mode
		if solicitud.TipoItinerario != nil {
			code := solicitud.TipoItinerario.Codigo
			switch code {
			case "SOLO_IDA":
				tmp := *solicitud
				ida = &tmp
			case "SOLO_VUELTA":
				tmp := *solicitud
				tmp.FechaIda = tmp.FechaVuelta
				tmp.Origen, tmp.Destino = tmp.Destino, tmp.Origen
				vuelta = &tmp
			default:
				// Round Trip (IDA_VUELTA) or default
				tmpIda := *solicitud
				ida = &tmpIda

				// Set vuelta only if it makes sense (e.g., date exists or we want to show it as available/empty?)
				// CupoReport logic implies we show segments even if empty? No, it uses if sol != nil.
				// If we have a round trip request, we should probably show the Vuelta segment even if Pending (date nil).
				// If date is nil, the table will show "-".

				tmpVuelta := *solicitud
				tmpVuelta.FechaIda = tmpVuelta.FechaVuelta
				tmpVuelta.Origen, tmpVuelta.Destino = tmpVuelta.Destino, tmpVuelta.Origen
				vuelta = &tmpVuelta
			}
		}
	}

	drawSegment("TRAYECTO DE IDA", ida)
	drawSegment("TRAYECTO DE VUELTA", vuelta)

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
