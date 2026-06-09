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

func (s *ReportService) GeneratePV01(ctx context.Context, solicitud *models.Solicitud, personaView *models.MongoPersonaView, mode string) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Logic to calculate deadline
	var masReciente time.Time
	for _, it := range solicitud.Items {
		if it.Fecha != nil {
			if masReciente.IsZero() || it.Fecha.After(masReciente) {
				masReciente = *it.Fecha
			}
		}
	}

	fechaLimiteStr := ""
	if !masReciente.IsZero() {
		limite := utils.CalcularFechaLimiteDescargo(masReciente)
		fechaLimiteStr = limite.Format("02/01/2006")
	}

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetX(3)
		if fechaLimiteStr != "" {
			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 0)
			pdf.CellFormat(0, 4.5, tr("PLAZO DE PRESENTACION DESCARGO HASTA EL: "+fechaLimiteStr), "", 1, "L", false, 0, "")
		}
		pdf.SetX(3)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("La solicitud deberá presentarse con 48 horas de anticipación (Art. 24). El descargo, adjuntando pases a bordo originales del/los tramo(s) utilizado(s), deberá efectuarse dentro de 8 días hábiles posteriores al retorno o finalización del tramo (Arts. 25 y 48); para pasajes internacionales, el plazo será de 5 días hábiles. No se recibirán solicitudes por derecho con descargos anteriores pendientes.\nEl incumplimiento dará lugar a sanciones conforme al Reglamento de Pasajes y Viáticos.")
		pdf.MultiCell(209, 2.5, disclaimer, "", "L", false)
		s.drawPageBorder(pdf)
	})

	pdf.AddPage()
	s.drawReportHeader(pdf, tr, "FORM-PV-01", "FORMULARIO DE SOLICITUD", "PASAJES AEREOS PARA SENADORAS Y SENADORES", "GESTION: "+solicitud.CreatedAt.Format("2006"), solicitud.Codigo)

	pdf.SetFont("Arial", "", 10)

	// Body
	s.drawLabelBox(pdf, tr, "NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 40, 150, false)
	s.drawLabelBox(pdf, tr, "C.I. :", solicitud.Usuario.CI, 40, 60, true)
	s.drawLabelBox(pdf, tr, "TEL. REF :", solicitud.Usuario.Phone, 30, 60, false)

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

	s.drawLabelBox(pdf, tr, "SENADOR POR EL DPTO. :", origenUser, 40, 60, true)

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

	s.drawLabelBox(pdf, tr, "UNIDAD FUNCIONAL :", unit, 40, 150, false)

	fechaSol := solicitud.CreatedAt.Format("02/01/2006")
	horaSol := solicitud.CreatedAt.Format("15:04")
	s.drawLabelBox(pdf, tr, "FECHA DE SOLICITUD :", fechaSol, 40, 60, true)
	s.drawLabelBox(pdf, tr, "HORA :", horaSol, 30, 60, false)

	concepto := ""
	if solicitud.TipoSolicitud != nil {
		concepto = solicitud.TipoSolicitud.Nombre
	}
	// Note: simplified logic compared to CupoReport which checks ConceptoViaje
	if solicitud.TipoSolicitud != nil && solicitud.TipoSolicitud.ConceptoViaje != nil {
		concepto = solicitud.TipoSolicitud.ConceptoViaje.Nombre
	}

	pdf.SetXY(10, pdf.GetY()+5)
	s.drawLabelBox(pdf, tr, "CONCEPTO DE VIAJE :", concepto, 40, 60, true)

	mesYNum := ""
	if strings.Contains(strings.ToUpper(concepto), "DERECHO") {
		if solicitud.CupoDerechoItem != nil {
			monthStr := utils.TranslateMonth(time.Month(solicitud.CupoDerechoItem.Mes))
			s := strings.ToUpper(solicitud.CupoDerechoItem.Semana)
			cupoStr := strings.Replace(s, "SEMANA", "CUPO", 1)
			if !strings.Contains(cupoStr, "CUPO") {
				cupoStr = "CUPO " + cupoStr
			}
			mesYNum = fmt.Sprintf("%s - %s", monthStr, cupoStr)
		}
	}
	s.drawLabelBox(pdf, tr, "MES DE VIAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	// SEGMENTS LOGIC
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

	aerolineaLabel := "-"
	if solicitud.Aerolinea != nil {
		if solicitud.Aerolinea.Sigla != "" {
			aerolineaLabel = solicitud.Aerolinea.Sigla
		} else {
			aerolineaLabel = solicitud.Aerolinea.Nombre
		}
	}

	pdf.Ln(2)
	s.drawSolicitudSegment(ctx, pdf, tr, "TRAYECTO DE IDA", idaItem, aerolineaLabel)
	pdf.Ln(4)
	s.drawSolicitudSegment(ctx, pdf, tr, "TRAYECTO DE VUELTA", vueltaItem, aerolineaLabel)

	pdf.Ln(8)

	authVal := solicitud.Autorizacion
	if authVal == "" {
		authVal = "PD"
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(40, 6, tr("Nro(Memo/RD/Nota JG/RC) :"), "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(150, 6, "  "+tr(authVal), "1", 1, "L", false, 0, "")

	pdf.Ln(2)

	sigY := pdf.GetY() + 20
	if sigY > 230 {
		pdf.AddPage()
		sigY = 40
	}
	s.drawSignatureBlock(pdf, tr, sigY, "SELLO UNIDAD SOLICITANTE", "", "", "FIRMA / SELLO SOLICITANTE", "", "")

	return pdf
}

// GeneratePV02 genera el formulario FORM-PV-02 para solicitud de pasajes oficial (funcionarios).
// personaView puede ser nil; si viene de MongoDB (por CI) se usan Cargo y Dependencia para el PDF.
func (s *ReportService) GeneratePV02(ctx context.Context, solicitud *models.Solicitud, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Logic to calculate deadline
	var masReciente time.Time
	for _, it := range solicitud.Items {
		if it.Fecha != nil {
			if masReciente.IsZero() || it.Fecha.After(masReciente) {
				masReciente = *it.Fecha
			}
		}
	}

	fechaLimiteStr := ""
	if !masReciente.IsZero() {
		limite := utils.CalcularFechaLimiteDescargo(masReciente)
		fechaLimiteStr = limite.Format("02/01/2006")
	}

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetX(3)
		if fechaLimiteStr != "" {
			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 0)
			pdf.CellFormat(0, 4.5, tr("PLAZO DE PRESENTACION DESCARGO HASTA EL: "+fechaLimiteStr), "", 1, "L", false, 0, "")
		}
		pdf.SetX(3)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("La solicitud deberá presentarse con 48 horas de anticipación (Art. 24). El descargo, adjuntando pases a bordo originales del/los tramo(s) utilizado(s), deberá efectuarse dentro de 8 días hábiles posteriores al retorno o finalización del tramo (Arts. 25 y 48); para pasajes internacionales, el plazo será de 5 días hábiles. No se recibirán solicitudes por derecho con descargos anteriores pendientes.\nEl incumplimiento dará lugar a sanciones conforme al Reglamento de Pasajes y Viáticos.")
		pdf.MultiCell(209, 2.5, disclaimer, "", "L", false)
		s.drawPageBorder(pdf)
	})

	pdf.AddPage()
	s.drawReportHeader(pdf, tr, "FORM-PV-02", "FORMULARIO DE SOLICITUD", "PASAJES AEREOS PARA FUNCIONARIOS(AS)", "GESTION: "+solicitud.CreatedAt.Format("2006"), solicitud.Codigo)

	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)

	// DATOS DEL SUSCRITO
	s.drawLabelBox(pdf, tr, "NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 45, 145, false)
	s.drawLabelBox(pdf, tr, "C.I. :", solicitud.Usuario.CI, 45, 55, true)
	s.drawLabelBox(pdf, tr, "TEL. REF :", solicitud.Usuario.Phone, 35, 55, false)

	cargoStr := ""
	if personaView != nil && personaView.Cargo != "" {
		cargoStr = personaView.Cargo
	} else if solicitud.Usuario.Cargo != nil {
		cargoStr = solicitud.Usuario.Cargo.Descripcion
	}
	s.drawLabelBox(pdf, tr, "CARGO :", cargoStr, 45, 145, false)

	unidadStr := ""
	if personaView != nil && personaView.Dependencia != "" {
		unidadStr = personaView.Dependencia
	} else if solicitud.Usuario.Oficina != nil {
		unidadStr = solicitud.Usuario.Oficina.Detalle
	}
	s.drawLabelBox(pdf, tr, "UNIDAD FUNCIONAL :", unidadStr, 45, 145, false)

	fechaSol := solicitud.CreatedAt.Format("02/01/2006")
	horaSol := solicitud.CreatedAt.Format("15:04")
	s.drawLabelBox(pdf, tr, "FECHA DE SOLICITUD :", fechaSol, 45, 55, true)
	s.drawLabelBox(pdf, tr, "HORA :", horaSol, 25, 40, false)

	authVal := solicitud.Autorizacion
	if authVal == "" {
		authVal = "PD"
	}
	s.drawLabelBox(pdf, tr, "Nro(Memo/RD/Nota JG/RC) :", authVal, 45, 55, false)

	conceptoViaje := "OFICIAL"
	if solicitud.TipoSolicitud != nil {
		conceptoViaje = "OFICIAL - " + strings.ToUpper(solicitud.TipoSolicitud.Nombre)
	}
	s.drawLabelBox(pdf, tr, "CONCEPTO DE VIAJE :", conceptoViaje, 45, 145, false)

	pdf.Ln(4)

	// OBJETIVO DEL VIAJE (motivo)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(190, 6, tr("OBJETIVO DEL VIAJE"), "L,R,T", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	motivo := solicitud.Motivo
	if motivo == "" {
		motivo = " "
	}
	pdf.MultiCell(190, 5, tr(motivo), "L,R,B", "L", false)

	pdf.Ln(5)

	// SOLICITA PASAJES DE IDA Y VUELTA EN LA SIGUIENTE RUTA
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr("SOLICITA PASAJES DE IDA Y VUELTA EN LA SIGUIENTE RUTA"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	// LINEA AEREA SUGERIDA + NACIONAL / INTERNACIONAL
	pdf.SetFont("Arial", "B", 8)

	esNacional := strings.ToUpper(solicitud.AmbitoViajeCodigo) == "NACIONAL"
	esInternacional := strings.ToUpper(solicitud.AmbitoViajeCodigo) == "INTERNACIONAL"

	pdf.SetX(145)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(18, 6, "NACIONAL", "", 0, "R", false, 0, "")
	xCheck, yCheck := pdf.GetX(), pdf.GetY()
	pdf.Rect(xCheck+1, yCheck+1, 4, 4, "D")
	if esNacional {
		pdf.Text(xCheck+1.5, yCheck+4.5, "X")
	}
	pdf.SetX(xCheck + 8)
	pdf.CellFormat(22, 6, "INTERNACIONAL", "", 0, "R", false, 0, "")
	xCheck, yCheck = pdf.GetX(), pdf.GetY()
	pdf.Rect(xCheck+1, yCheck+1, 4, 4, "D")
	if esInternacional {
		pdf.Text(xCheck+1.5, yCheck+4.5, "X")
	}
	pdf.Ln(8)

	var itemsIda, itemsVuelta []models.SolicitudItem
	for _, item := range solicitud.Items {
		switch item.Tipo {
		case models.TipoSolicitudItemIda:
			itemsIda = append(itemsIda, item)
		case models.TipoSolicitudItemVuelta:
			itemsVuelta = append(itemsVuelta, item)
		}
	}

	drawRutaSeccion := func(titulo string, items []models.SolicitudItem) {
		if len(items) == 0 {
			return
		}
		primero := items[0]
		ultimo := items[len(items)-1]
		origenStr := primero.OrigenIATA
		if primero.Origen != nil {
			origenStr = primero.Origen.Ciudad
		}
		destStr := ultimo.DestinoIATA
		if ultimo.Destino != nil {
			destStr = ultimo.Destino.Ciudad
		}
		rutaResumen := origenStr + " - " + destStr

		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+titulo), "B", 1, "L", false, 0, "")
		pdf.Ln(1)

		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(35, 6, tr("Ruta :"), "", 0, "R", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(155, 6, "  "+tr(rutaResumen), "1", 1, "L", false, 0, "")
		pdf.Ln(2)

		// Tabla mismo formato que PV-01: FECHA/HORA SOLICITUD, ESTADO, AEROLÍNEA, RUTA, FECHA VIAJE, HORA VIAJE
		fechaSol := solicitud.CreatedAt.Format("02/01/2006")
		horaSol := solicitud.CreatedAt.Format("15:04")

		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(32, 8, tr("FECHA/HORA SOLICITUD"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 8, tr("AEROLÍNEA SUGERIDA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(52, 8, tr("RUTA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 8, tr("FECHA VIAJE"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(18, 8, tr("HORA VIAJE"), "1", 1, "C", true, 0, "")

		for _, item := range items {
			origenStr := item.OrigenIATA
			if item.Origen != nil {
				origenStr = item.Origen.Ciudad
			}
			destStr := item.DestinoIATA
			if item.Destino != nil {
				destStr = item.Destino.Ciudad
			}
			rut := fmt.Sprintf("%s  -  %s", origenStr, destStr)

			fecha := "-"
			hora := "-"
			if item.Fecha != nil {
				fecha = utils.FormatDateShortES(*item.Fecha)
				if len(fecha) > 5 {
					fecha = fecha[:len(fecha)-5]
				}
				hora = item.Fecha.Format("15:04")
			}

			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(32, 8, fmt.Sprintf("%s %s", fechaSol, horaSol), "1", 0, "C", false, 0, "")

			// Aerolínea por tramo
			aerolineaItem := "-"
			if item.Aerolinea != nil {
				if item.Aerolinea.Sigla != "" {
					aerolineaItem = item.Aerolinea.Sigla
				} else {
					aerolineaItem = item.Aerolinea.Nombre
				}
			}

			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 128)
			pdf.CellFormat(25, 8, tr(item.GetEstado()), "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 7)
			pdf.CellFormat(35, 8, tr(aerolineaItem), "1", 0, "C", false, 0, "")
			pdf.CellFormat(52, 8, tr(rut), "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(30, 8, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 8, hora, "1", 1, "C", false, 0, "")
		}
		pdf.Ln(5)
	}

	drawRutaSeccion("RUTA DE IDA", itemsIda)
	drawRutaSeccion("RUTA DE VUELTA", itemsVuelta)

	sigY := pdf.GetY() + 20
	if sigY > 230 {
		pdf.AddPage()
		sigY = 40
	}
	s.drawSignatureBlock(pdf, tr, sigY, "SELLO UNIDAD SOLICITANTE", "", "", "FIRMA / SELLO SOLICITANTE", "", "")

	pdf.SetFont("Arial", "I", 8)
	return pdf
}

func (s *ReportService) drawSolicitudSegment(_ context.Context, pdf *gofpdf.Fpdf, tr func(string) string, title string, item *models.SolicitudItem, aerolineaSugerida string) {
	if item == nil {
		return
	}
	const totalWidth = 200.0
	const startX = 3.0

	pdf.SetX(startX)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(totalWidth, 6, tr(" "+title), "B", 1, "L", false, 0, "")

	if item.GetEstado() != "PENDIENTE" {
		fecha := "-"
		hora := "-"
		if item.Fecha != nil {
			fecha = utils.FormatDateShortES(*item.Fecha)
			if len(fecha) > 5 {
				fecha = fecha[:len(fecha)-5] // Remove year
			}
			hora = item.Fecha.Format("15:04")
		}

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
		pdf.SetX(startX)
		pdf.SetFont("Arial", "B", 7)
		pdf.SetFillColor(245, 245, 245)

		// Column Widths (Total: 200)
		w1, w2, w3, w4, w5, w6 := 35.0, 25.0, 40.0, 60.0, 30.0, 20.0

		// Headers
		pdf.CellFormat(w1, 8, tr("FECHA/HORA REG."), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w2, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w3, 8, tr("AEROLÍNEA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w4, 8, tr("RUTA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w5, 8, tr("FECHA VIAJE"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(w6, 8, tr("HORA"), "1", 1, "C", true, 0, "")

		// Data Row
		pdf.SetX(startX)
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(w1, 8, item.CreatedAt.Format("02/01/2006 15:04"), "1", 0, "C", false, 0, "")

		pdf.SetFont("Arial", "B", 7)
		pdf.SetTextColor(0, 0, 128)
		pdf.CellFormat(w2, 8, tr(item.GetEstado()), "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "", 7)

		aerolineaNombre := aerolineaSugerida
		if item.Aerolinea != nil {
			if item.Aerolinea.Sigla != "" {
				aerolineaNombre = item.Aerolinea.Sigla
			} else {
				aerolineaNombre = item.Aerolinea.Nombre
			}
		}

		pdf.CellFormat(w3, 8, tr(aerolineaNombre), "1", 0, "C", false, 0, "")
		pdf.CellFormat(w4, 8, tr(rut), "1", 0, "C", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(w5, 8, tr(fecha), "1", 0, "C", false, 0, "")
		pdf.CellFormat(w6, 8, hora, "1", 1, "C", false, 0, "")

	} else {
		pdf.SetX(startX)
		pdf.SetFont("Arial", "I", 9)
		pdf.CellFormat(totalWidth, 10, tr(" TRAMO NO SOLICITADO"), "", 1, "C", false, 0, "")
	}
	pdf.Ln(2)
}
