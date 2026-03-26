package services

import (
	"bytes"
	"context"
	"fmt"

	"os"
	"os/exec"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

type ReportService struct {
	solicitudRepo *repositories.SolicitudRepository
	aerolineaRepo *repositories.AerolineaRepository
	pasajeRepo    *repositories.PasajeRepository
	agenciaRepo   *repositories.AgenciaRepository
	configService *ConfiguracionService
}

func NewReportService(
	solicitudRepo *repositories.SolicitudRepository,
	aerolineaRepo *repositories.AerolineaRepository,
	pasajeRepo *repositories.PasajeRepository,
	agenciaRepo *repositories.AgenciaRepository,
	configService *ConfiguracionService,
) *ReportService {
	return &ReportService{
		solicitudRepo: solicitudRepo,
		aerolineaRepo: aerolineaRepo,
		pasajeRepo:    pasajeRepo,
		agenciaRepo:   agenciaRepo,
		configService: configService,
	}
}

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

	pdf.Ln(2)
	s.drawSolicitudSegment(ctx, pdf, tr, "TRAYECTO DE IDA", idaItem, solicitud.AerolineaSugerida)
	pdf.Ln(4)
	s.drawSolicitudSegment(ctx, pdf, tr, "TRAYECTO DE VUELTA", vueltaItem, solicitud.AerolineaSugerida)

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

	pdf.SetY(220)

	pdf.SetY(220)
	s.drawSignatureBlock(pdf, tr, 230, "SELLO UNIDAD SOLICITANTE", "FIRMA / SELLO SOLICITANTE", "")

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

	s.drawLabelBox(pdf, tr, "CONCEPTO DE VIAJE :", "OFICIAL", 45, 55, false)

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
	aerolineaNombre := solicitud.AerolineaSugerida
	if aerolineaNombre == "" {
		aerolineaNombre = "-"
	} else if aerolinea, err := s.aerolineaRepo.FindByID(ctx, aerolineaNombre); err == nil {
		if aerolinea.Sigla != "" {
			aerolineaNombre = aerolinea.Sigla
		} else {
			aerolineaNombre = aerolinea.Nombre
		}
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(45, 6, tr("LINEA AEREA SUGERIDA :"), "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(80, 6, "  "+tr(aerolineaNombre), "1", 0, "L", false, 0, "")

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
		aerolineaNombre := solicitud.AerolineaSugerida
		if aerolineaNombre != "" {
			if aerolinea, err := s.aerolineaRepo.FindByID(ctx, solicitud.AerolineaSugerida); err == nil {
				if aerolinea.Sigla != "" {
					aerolineaNombre = aerolinea.Sigla
				} else {
					aerolineaNombre = aerolinea.Nombre
				}
			}
		} else {
			aerolineaNombre = "-"
		}

		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(32, 8, tr("FECHA/HORA SOLICITUD"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 8, tr("ESTADO"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 8, tr("AEROLÍNEA"), "1", 0, "C", true, 0, "")
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
			rut := fmt.Sprintf("%s  >>  %s", origenStr, destStr)

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

			pdf.SetFont("Arial", "B", 7)
			pdf.SetTextColor(0, 0, 128)
			pdf.CellFormat(25, 8, tr(item.GetEstado()), "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 7)
			pdf.CellFormat(35, 8, tr(aerolineaNombre), "1", 0, "C", false, 0, "")
			pdf.CellFormat(52, 8, tr(rut), "1", 0, "C", false, 0, "")
			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(30, 8, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 8, hora, "1", 1, "C", false, 0, "")
		}
		pdf.Ln(5)
	}

	drawRutaSeccion("RUTA DE IDA", itemsIda)
	drawRutaSeccion("RUTA DE VUELTA", itemsVuelta)

	pdf.SetY(240)
	s.drawSignatureBlock(pdf, tr, 248, "SELLO UNIDAD SOLICITANTE", "FIRMA / SELLO SOLICITANTE", "")

	pdf.SetFont("Arial", "I", 8)
	return pdf
}

func (s *ReportService) GeneratePV05(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	solicitud := descargo.Solicitud

	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("NO SE DARA CURSO AL TRAMITE DE PASAJES Y VIATICOS, EN CASO DE VERIFICARSE QUE LOS DOCUMENTOS DE DECLARATORIA Y/O DESCARGO EN COMISION OFICIAL PRESENTEN ALTERACIONES, BORRONES O ENMIENDAS QUE MODIFIQUEN EL CONTENIDO DE LA MISMA, SIN PERJUICIO DE INICIAR LAS ACCIONES LEGALES QUE CORRESPONDA.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
		s.drawPageBorder(pdf)
	})

	pdf.AddPage()
	s.drawReportHeader(pdf, tr, "FORM-PV-05", "FORMULARIO DE DESCARGO", "PASAJES AEREOS PARA SENADORAS Y SENADORES", "GESTION: "+solicitud.CreatedAt.Format("2006"), solicitud.Codigo)

	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)

	s.drawLabelBox(pdf, tr, "NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 40, 150, false)
	s.drawLabelBox(pdf, tr, "C.I. :", solicitud.Usuario.CI, 40, 60, true)
	s.drawLabelBox(pdf, tr, "TEL. REF :", solicitud.Usuario.Phone, 30, 60, false)

	origenUser := ""
	if solicitud.Usuario.Origen != nil {
		origenUser = solicitud.Usuario.Origen.Ciudad
	}
	tipoUsuario := solicitud.Usuario.Tipo
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

	mesLabel := mes
	if solicitud.CupoDerechoItem != nil {
		cupoNum := strings.Replace(solicitud.CupoDerechoItem.Semana, "SEMANA", "CUPO", 1)
		mesLabel = fmt.Sprintf("%s - %s", mes, cupoNum)
	}

	s.drawLabelBox(pdf, tr, "CORRESPONDIENTE AL MES DE :", mesLabel, 50, 60, false)
	pdf.Ln(2)

	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("DESCARGO DE PASAJES (ADJUNTAR PASES A BORDO)"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	drawSubTable := func(subTitle string, headerBoleto string, rows []models.DetalleItinerarioDescargo) {
		if len(rows) == 0 {
			return
		}
		if subTitle != "" {
			pdf.SetFillColor(240, 240, 240)
			pdf.SetFont("Arial", "B", 8)
			pdf.CellFormat(190, 5, tr(subTitle), "1", 1, "C", true, 0, "")
		}
		pdf.SetFillColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(70, 5, tr("RUTA"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 5, tr("FECHA DE VIAJE"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 5, tr(headerBoleto), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 5, tr("N° PASE A BORDO"), "1", 1, "C", false, 0, "")

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

			paseVal := r.NumeroPaseAbordo
			if r.EsDevolucion {
				pdf.SetTextColor(200, 0, 0)
				pdf.SetFont("Arial", "B", 7)
				paseVal = "DEVUELTO"
			}
			pdf.CellFormat(40, 6, tr(paseVal), "1", 1, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 8)
		}
	}

	// Group by Ticket consolidate devo/mod status
	type TicketRepGroup struct {
		Boleto       string
		Detalles     []models.DetalleItinerarioDescargo
		EsDevolucion bool
	}
	ticketsMap := make(map[string]*TicketRepGroup)
	var ticketsOrder []string

	for _, d := range descargo.DetallesItinerario {
		key := d.Boleto
		if key == "" {
			key = "SN-" + d.Ruta
		}
		if _, ok := ticketsMap[key]; !ok {
			ticketsMap[key] = &TicketRepGroup{Boleto: d.Boleto}
			ticketsOrder = append(ticketsOrder, key)
		}
		if d.EsDevolucion {
			ticketsMap[key].EsDevolucion = true
		}
		ticketsMap[key].Detalles = append(ticketsMap[key].Detalles, d)
	}

	drawSegmentBlock := func(title string, typeOrig, typeRepro models.TipoDetalleItinerario, tMap map[string]*TicketRepGroup, tOrder []string) {
		var origRows, reproRows []models.DetalleItinerarioDescargo

		for _, key := range tOrder {
			g := tMap[key]
			for _, d := range g.Detalles {
				// Propagate group return status to individual segment for display
				if g.EsDevolucion {
					d.EsDevolucion = true
				}
				switch d.Tipo {
				case typeOrig:
					origRows = append(origRows, d)
				case typeRepro:
					reproRows = append(reproRows, d)
				}
			}
		}

		if len(origRows) > 0 || len(reproRows) > 0 {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(190, 6, tr(" "+title), "", 1, "L", false, 0, "")
			pdf.Ln(1)
			drawSubTable("", "N° BOLETO ORIGINAL", origRows)
			if len(reproRows) > 0 {
				pdf.Ln(1)
				drawSubTable("REPROGRAMACIÓN", "N° BOLETO REPROGRAMADO", reproRows)
			}
		}
	}

	drawSegmentBlock("TRAMO DE IDA", models.TipoDetalleIdaOriginal, models.TipoDetalleIdaReprogramada, ticketsMap, ticketsOrder)
	pdf.Ln(4)
	drawSegmentBlock("TRAMO DE RETORNO", models.TipoDetalleVueltaOriginal, models.TipoDetalleVueltaReprogramada, ticketsMap, ticketsOrder)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" PASAJE ABIERTO-OPEN TICKET"), "B", 1, "L", false, 0, "")

	hasReturns := false
	for _, key := range ticketsOrder {
		g := ticketsMap[key]
		if g.EsDevolucion {
			hasReturns = true
			tipoStr := "TRAMO"
			if len(g.Detalles) > 0 {
				if strings.Contains(string(g.Detalles[0].Tipo), "IDA") {
					tipoStr = "TRAMO DE IDA"
				} else {
					tipoStr = "TRAMO DE RETORNO"
				}
			}

			// Find cost and full route from Pasaje
			cost := 0.0
			fullRoute := ""
			if descargo.Solicitud != nil {
				for _, item := range descargo.Solicitud.Items {
					for _, p := range item.Pasajes {
						if p.NumeroBoleto == g.Boleto && g.Boleto != "" {
							cost = p.Costo
							fullRoute = p.Ruta
							break
						}
					}
				}
			}

			// Fallback route reconstruction if not found in Pasaje
			if fullRoute == "" && len(g.Detalles) > 0 {
				var routeParts []string
				for i, det := range g.Detalles {
					parts := strings.Split(det.Ruta, "-")
					if i == 0 {
						routeParts = append(routeParts, strings.TrimSpace(parts[0]))
					}
					if len(parts) >= 2 {
						routeParts = append(routeParts, strings.TrimSpace(parts[1]))
					}
				}
				fullRoute = strings.Join(routeParts, " - ")
			}

			s.drawReturnTableSummarized(pdf, tr, tipoStr, g.Boleto, fullRoute, cost)
			pdf.Ln(1)
		}
	}

	if !hasReturns {
		s.drawReturnTableSummarized(pdf, tr, "", "", "", 0)
	}
	pdf.Ln(10)

	sigY := 220.0
	pdf.SetY(sigY)
	s.drawSignatureBlock(pdf, tr, sigY+10, "SELLO UNIDAD SOLICITANTE", "FIRMA/RESPONSABLE PRESENTACION DEL DESCARGO", "")

	return pdf
}

func (s *ReportService) GeneratePV06(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	solicitud := descargo.Solicitud

	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("NO SE DARA CURSO AL TRAMITE DE PASAJES Y VIATICOS, EN CASO DE VERIFICARSE QUE LOS DOCUMENTOS DE DECLARATORIA Y/O DESCARGO EN COMISION OFICIAL PRESENTEN ALTERACIONES, BORRONES O ENMIENDAS QUE MODIFIQUEN EL CONTENIDO DE LA MISMA, SIN PERJUICIO DE INICIAR LAS ACCIONES LEGALES QUE CORRESPONDA.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
		s.drawPageBorder(pdf)
	})

	pdf.AddPage()
	s.drawReportHeader(pdf, tr, "FORM-PV-06", "FORMULARIO DE DESCARGO", "PASAJES OFICIALES", "SERVIDORES PÚBLICOS - GESTION: "+solicitud.CreatedAt.Format("2006"), solicitud.Codigo)

	pdf.SetFont("Arial", "", 10)

	cargoStr := ""
	if personaView != nil && personaView.Cargo != "" {
		cargoStr = utils.CleanName(personaView.Cargo)
	} else if solicitud.Usuario.Cargo != nil {
		cargoStr = utils.CleanName(solicitud.Usuario.Cargo.Descripcion)
	}

	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)

	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)

	aQuien := ""
	if descargo.Oficial != nil {
		aQuien = descargo.Oficial.DirigidoA
	}

	if aQuien == "" && solicitud.Usuario.Encargado != nil {
		aQuien = solicitud.Usuario.Encargado.GetNombreCompleto()
		if solicitud.Usuario.Encargado.Cargo != nil {
			aQuien += " - " + solicitud.Usuario.Encargado.Cargo.Descripcion
		}
	}

	s.drawMemoRow(pdf, tr, "A :", aQuien)
	s.drawMemoRow(pdf, tr, "De :", solicitud.Usuario.GetNombreCompleto())
	s.drawMemoRow(pdf, tr, "Cargo :", cargoStr)
	s.drawMemoRow(pdf, tr, "Fecha :", solicitud.CreatedAt.Format("02/01/2006"))

	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("Datos Generales del Viaje:"), "", 1, "L", false, 0, "")

	// Row for "Lugar del Viaje"
	pdf.SetFillColor(255, 255, 255) // Peach color from image
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(35, 8, tr("Lugar del Viaje:"), "1", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 10)
	lugarViaje := "-"
	// Search for the LAST item of type IDA to get the final destination
	for i := len(solicitud.Items) - 1; i >= 0; i-- {
		if solicitud.Items[i].Tipo == models.TipoSolicitudItemIda {
			if solicitud.Items[i].Destino != nil {
				lugarViaje = solicitud.Items[i].Destino.Ciudad
			}
			break
		}
	}
	if lugarViaje == "-" {
		lugarViaje = solicitud.GetDestinoCiudad()
	}

	pdf.CellFormat(155, 8, "  "+tr(lugarViaje), "1", 1, "L", false, 0, "")

	pdf.Ln(2)

	// Row for "Nº de Memorándum o Resolución"
	nroMemo := ""
	if descargo.Oficial != nil {
		nroMemo = descargo.Oficial.NroMemorandum
	}

	wLabelMemo := 65.0
	wValueMemo := 50.0
	totalW := wLabelMemo + wValueMemo
	currentX := pdf.GetX()
	pdf.SetX(currentX + (190-totalW)/2) // Centering the block in 190mm width

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(wLabelMemo, 7, tr("N° de Memorándum o Resolución:"), "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(wValueMemo, 7, tr(nroMemo), "1", 1, "L", false, 0, "")

	pdf.Ln(2)
	pdf.Ln(2)

	// INFORME DETALLADO PV-06
	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("INFORME DE COMISIÓN"), "B", 1, "C", false, 0, "")
	pdf.Ln(4)

	// Memo y Objetivo
	if nroMemo == "" {
		nroMemo = "S/N"
	}
	objViaje := "No especificado"
	if descargo.Oficial != nil {
		if descargo.Oficial.NroMemorandum != "" {
			nroMemo = descargo.Oficial.NroMemorandum
		}
		if descargo.Oficial.ObjetivoViaje != "" {
			objViaje = descargo.Oficial.ObjetivoViaje
		}
	}

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(40, 6, tr("OBJETIVO DEL VIAJE:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(150, 6, tr(objViaje), "", "L", false)
	pdf.Ln(4)

	// Actividades
	infActividades := "Sin información"
	if descargo.Oficial != nil && descargo.Oficial.InformeActividades != "" {
		infActividades = descargo.Oficial.InformeActividades
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr("1. ACTIVIDADES REALIZADAS:"), "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.SetX(15) // Sangría
	pdf.MultiCell(175, 5, tr(infActividades), "", "J", false)
	pdf.Ln(4)

	// Resultados
	if descargo.Oficial != nil && descargo.Oficial.ResultadosViaje != "" {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr("2. RESULTADOS OBTENIDOS:"), "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.SetX(15)
		pdf.MultiCell(175, 5, tr(descargo.Oficial.ResultadosViaje), "", "J", false)
		pdf.Ln(4)
	}

	// Conclusiones
	if descargo.Oficial != nil && descargo.Oficial.ConclusionesRecomendaciones != "" {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr("3. CONCLUSIONES Y RECOMENDACIONES:"), "", 1, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.SetX(15)
		pdf.MultiCell(175, 5, tr(descargo.Oficial.ConclusionesRecomendaciones), "", "J", false)
		pdf.Ln(4)
	}

	// Anexos (Imágenes) - Parte del Informe
	if descargo.Oficial != nil && len(descargo.Oficial.Anexos) > 0 {
		const (
			pageWidth     = 190.0
			pageHeight    = 255.0 // Conservative limit before footer
			marginSide    = 10.0
			imgSpacing    = 4.0
			rowSpacing    = 10.0
			maxRowHeight  = 90.0 // Limit to avoid huge images
			minImageWidth = 55.0 // Allow up to 3 columns (~60mm each)
		)

		// Check initial space for title
		if pdf.GetY() > (pageHeight - 40.0) {
			pdf.AddPage()
		}

		pdf.Ln(4)
		pdf.SetX(marginSide)
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(190, 8, tr("5. Anexos (Reseña fotográfica)"), "B", 1, "C", false, 0, "")
		pdf.Ln(6)

		anexos := descargo.Oficial.Anexos
		i := 0
		for i < len(anexos) {
			// We'll try to build a row of images
			var rowIndices []int
			currentRowWidth := 0.0
			//rowMaxH := 0.0 // This variable was declared but not used in the new logic, can be removed or ignored.

			// Greedy approach to fill a row
			for j := i; j < len(anexos) && len(rowIndices) < 3; j++ {
				anexo := anexos[j]
				if _, err := os.Stat(anexo.Archivo); err != nil {
					continue
				}
				pdf.RegisterImage(anexo.Archivo, "")
				info := pdf.GetImageInfo(anexo.Archivo)
				if info == nil || info.Height() == 0 {
					continue
				}

				aspect := info.Width() / info.Height()
				// Estimate width if we fit to a reasonable height
				// If we aim for ~70mm height:
				estW := 70.0 * aspect
				if estW > 180.0 {
					estW = 180.0
				}
				if estW < minImageWidth {
					estW = minImageWidth
				}

				if len(rowIndices) > 0 && (currentRowWidth+imgSpacing+estW) > pageWidth {
					break // Row is full
				}

				rowIndices = append(rowIndices, j)
				currentRowWidth += estW
				if len(rowIndices) > 1 {
					currentRowWidth += imgSpacing
				}
			}

			if len(rowIndices) == 0 {
				i++
				continue
			}

			// Now calculate actual widths and a common height for the row
			// to keep things aligned. We'll use a target height that fits the row into pageWidth.
			totalAspect := 0.0
			for _, idx := range rowIndices {
				info := pdf.GetImageInfo(anexos[idx].Archivo)
				totalAspect += info.Width() / info.Height()
			}

			// Target Width = PageWidth - spacing
			targetW := pageWidth - float64(len(rowIndices)-1)*imgSpacing
			// Height for the row to fill targetW
			h := targetW / totalAspect
			if h > maxRowHeight {
				h = maxRowHeight
			}

			// Check for page break
			if pdf.GetY()+h+rowSpacing > pageHeight {
				pdf.AddPage()
				pdf.SetY(15.0)
				pdf.SetFont("Arial", "I", 8)
				pdf.CellFormat(0, 6, tr("5. Anexos (Reseña fotográfica) (Continuación)"), "", 1, "R", false, 0, "")
				pdf.Ln(4)
			}

			// Draw the row
			startX := marginSide + (pageWidth-(h*totalAspect+float64(len(rowIndices)-1)*imgSpacing))/2.0
			currX := startX
			currY := pdf.GetY()
			for _, idx := range rowIndices {
				info := pdf.GetImageInfo(anexos[idx].Archivo)
				asp := info.Width() / info.Height()
				w := h * asp
				pdf.Image(anexos[idx].Archivo, currX, currY, w, 0, false, "", 0, "")
				currX += w + imgSpacing
			}

			pdf.SetY(currY + h + rowSpacing)
			i += len(rowIndices)
		}
	}

	// --- Nueva Página para Tramos e Itinerario ---
	pdf.AddPage()

	// Transporte y Devolución
	tipoTrans := "No especificado"
	if descargo.Oficial != nil && descargo.Oficial.TipoTransporte != "" {
		tipoTrans = descargo.Oficial.TipoTransporte
		if descargo.Oficial.PlacaVehiculo != "" {
			tipoTrans += " (Placa: " + descargo.Oficial.PlacaVehiculo + ")"
		}
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(50, 6, tr("4. TRANSPORTE UTILIZADO:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(140, 6, tr(tipoTrans), "", 1, "L", false, 0, "")

	pdf.Ln(6)

	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("DESCARGO DE PASAJES (ADJUNTAR PASES A BORDO)"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	s.drawSegmentBlock(pdf, tr, descargo, "TRAMO DE IDA", models.TipoDetalleIdaOriginal, models.TipoDetalleIdaReprogramada)
	pdf.Ln(4)
	s.drawSegmentBlock(pdf, tr, descargo, "TRAMO DE RETORNO", models.TipoDetalleVueltaOriginal, models.TipoDetalleVueltaReprogramada)
	pdf.Ln(4)

	if len(solicitud.Viaticos) > 0 {
		s.drawViaticosTable(pdf, tr, solicitud.Viaticos)
	}

	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(210, 6, tr(" DEVOLUCIÓN DE PASAJES (si aplica)"), "B", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(190, 5, tr(" (En caso de no haber utilizado el boleto emitido o un tramo informar en el siguiente cuadro)"), "", 1, "L", false, 0, "")

	// Group by Ticket consolidate devo/mod status
	type TicketRepGroup struct {
		Boleto       string
		Detalles     []models.DetalleItinerarioDescargo
		EsDevolucion bool
	}
	ticketsMap := make(map[string]*TicketRepGroup)
	var ticketsOrder []string

	for _, d := range descargo.DetallesItinerario {
		key := d.Boleto
		if key == "" {
			key = "SN-" + d.Ruta
		}
		if _, ok := ticketsMap[key]; !ok {
			ticketsMap[key] = &TicketRepGroup{Boleto: d.Boleto}
			ticketsOrder = append(ticketsOrder, key)
		}
		if d.EsDevolucion {
			ticketsMap[key].EsDevolucion = true
		}
		ticketsMap[key].Detalles = append(ticketsMap[key].Detalles, d)
	}

	hasReturns := false
	for _, key := range ticketsOrder {
		g := ticketsMap[key]
		if g.EsDevolucion {
			hasReturns = true
			tipoStr := "TRAMO"
			if len(g.Detalles) > 0 {
				if strings.Contains(string(g.Detalles[0].Tipo), "IDA") {
					tipoStr = "TRAMO DE IDA"
				} else {
					tipoStr = "TRAMO DE RETORNO"
				}
			}

			// Find cost and full route from Pasaje
			cost := 0.0
			fullRoute := ""
			if descargo.Solicitud != nil {
				for _, item := range descargo.Solicitud.Items {
					for _, p := range item.Pasajes {
						if p.NumeroBoleto == g.Boleto && g.Boleto != "" {
							cost = p.Costo
							fullRoute = p.Ruta
							break
						}
					}
				}
			}
			if fullRoute == "" && len(g.Detalles) > 0 {
				var routeParts []string
				for i, det := range g.Detalles {
					parts := strings.Split(det.Ruta, "-")
					if i == 0 {
						routeParts = append(routeParts, strings.TrimSpace(parts[0]))
					}
					if len(parts) >= 2 {
						routeParts = append(routeParts, strings.TrimSpace(parts[1]))
					}
				}
				fullRoute = strings.Join(routeParts, " - ")
			}

			s.drawReturnTableSummarized(pdf, tr, tipoStr, g.Boleto, fullRoute, cost)
			pdf.Ln(1)
		}
	}

	if !hasReturns {
		s.drawReturnTableSummarized(pdf, tr, "", "", "", 0)
	}

	pdf.Ln(5)

	// --- Sección de Devolución Estilizada (según imagen) ---
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 6, tr("En caso de Devolución de Pasajes y/o Viáticos:"), "", 1, "L", false, 0, "")

	bancoCuenta := s.configService.GetValue(ctx, "BANCO_CUENTA_DEVOLUCION")
	if bancoCuenta == "" {
		bancoCuenta = "10000005588211"
	}
	bancoNombre := s.configService.GetValue(ctx, "BANCO_NOMBRE_DEVOLUCION")
	if bancoNombre == "" {
		bancoNombre = "BANCO UNIÓN S.A."
	}

	startY := pdf.GetY()
	// Celda Izquierda: Información de la cuenta
	pdf.SetFont("Arial", "B", 9)
	infoCuenta := "Monto depositado en Bs. en la CUT – Cuenta Única del Tesoro, Código N° 3987069001, Libreta N° 00099021001."

	// Dibujar rectángulos (Fondo blanco/transparente)
	pdf.Rect(10, startY, 100, 25, "D")
	pdf.SetXY(12, startY+4)
	pdf.MultiCell(96, 5, tr(infoCuenta), "", "L", false)

	// Celda Derecha: Boleta
	pdf.SetXY(110, startY)
	pdf.Rect(110, startY, 90, 25, "D")

	// Título Boleta
	pdf.SetXY(112, startY+4)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(86, 6, tr("N° de Boleta de Depósito:"), "B", 1, "L", false, 0, "")

	// Valor Boleta (si existe)
	if descargo.Oficial != nil && descargo.Oficial.NroBoletaDeposito != "" {
		pdf.SetXY(112, startY+12)
		pdf.SetFont("Courier", "B", 12)
		pdf.CellFormat(86, 8, tr(descargo.Oficial.NroBoletaDeposito), "", 1, "C", false, 0, "")
	}

	pdf.SetXY(10, startY+30)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 10, tr("Es cuanto se informa para fines consiguientes."), "", 1, "L", false, 0, "")

	pdf.Ln(4)

	// --- Signatures (Dynamic Position) ---
	sigY := pdf.GetY() + 15

	// Ensure we don't start signatures too low on the page
	if sigY > 230 {
		pdf.AddPage()
		sigY = 40
	}

	pdf.SetY(sigY)

	if solicitud.Usuario.IsSenador() {
		s.drawSignatureBlock(pdf, tr, sigY+10, "SELLO UNIDAD SOLICITANTE", "FIRMA Y SELLO SENADOR(A)", "")
	} else {
		s.drawSignatureBlock(pdf, tr, sigY+10, "FIRMA Y SELLO SERVIDOR PÚBLICO", "Vo.Bo. Inmediato Superior", "")
	}

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
		return nil, fmt.Errorf("error generating base PDF: %w", err)
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
		data, err := os.ReadFile(tmpBase.Name())
		if err != nil {
			return nil, fmt.Errorf("error reading base PDF: %w", err)
		}
		return data, nil
	}

	// 4. Unir usando pdftk
	tmpFinal, err := os.CreateTemp("", "pv05_final_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFinal.Name())

	args := append(filesToMerge, "cat", "output", tmpFinal.Name())
	cmd := exec.CommandContext(ctx, "pdftk", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error merging PDFs with pdftk: %w - %s", err, stderr.String())
	}

	data, err := os.ReadFile(tmpFinal.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading final merged PDF: %w", err)
	}
	return data, nil
}

func (s *ReportService) GeneratePV06Complete(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) ([]byte, error) {
	// 1. Generar el PDF Base PV-06
	pdf := s.GeneratePV06(ctx, descargo, personaView)

	// Crear archivos temporales para la unión
	tmpBase, err := os.CreateTemp("", "pv06_base_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBase.Name())

	if err := pdf.OutputFileAndClose(tmpBase.Name()); err != nil {
		return nil, fmt.Errorf("error generating base PDF: %w", err)
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
		data, err := os.ReadFile(tmpBase.Name())
		if err != nil {
			return nil, fmt.Errorf("error reading base PDF: %w", err)
		}
		return data, nil
	}

	// 4. Unir usando pdftk
	tmpFinal, err := os.CreateTemp("", "pv06_final_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFinal.Name())

	args := append(filesToMerge, "cat", "output", tmpFinal.Name())
	cmd := exec.CommandContext(ctx, "pdftk", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error merging PDFs with pdftk: %w - %s", err, stderr.String())
	}

	data, err := os.ReadFile(tmpFinal.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading final merged PDF: %w", err)
	}
	return data, nil
}

func (s *ReportService) drawLabelBox(pdf *gofpdf.Fpdf, tr func(string) string, label, value string, wLabel, wBox float64, sameLine bool) {
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

func (s *ReportService) drawReportHeader(pdf *gofpdf.Fpdf, tr func(string) string, formCode, title, subtitle, gestion, code string) {
	// Adjusting based on border starting at X=3 and ending at X=213 (width 210)
	yBase := 8.0

	// 1. Logo Position (Right) - Now clearly inside the frame
	pdf.Image("web/static/img/logo_senado.png", 185, 4, 21, 0, false, "", 0, "")

	// 2. Left Block (Form Code / solicitud Code) - Starting at X=8 (5mm inside border)
	pdf.SetXY(6, yBase)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(45, 7, formCode, "", 2, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(45, 6, code, "", 0, "C", false, 0, "")

	// 3. Central Block (Titles) - Mathematically centered on the page (Letter width ~215.9)
	pdf.SetXY(0, yBase-1)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(215.9, 8, tr(title), "", 2, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(215.9, 5, tr(subtitle), "", 2, "C", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(215.9, 5, tr(gestion), "", 0, "C", false, 0, "")

	// 4. Horizontal Separator - Matching the exact width of top border
	pdf.SetLineWidth(0.3)
	pdf.Line(3, yBase+18, 213, yBase+18) // Aligned with the bottom of the header box

	pdf.SetY(yBase + 24)
}

func (s *ReportService) drawMemoRow(pdf *gofpdf.Fpdf, tr func(string) string, label, value string) {
	h := 7.0
	pdf.SetFillColor(255, 255, 255) // White background
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(30, h, tr(label), "1", 0, "R", true, 0, "")

	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(160, h, "  "+tr(value), "1", 1, "L", false, 0, "")
}

func (s *ReportService) drawSubTable(pdf *gofpdf.Fpdf, tr func(string) string, subTitle string, headerBoleto string, rows []models.DetalleItinerarioDescargo) {
	if subTitle != "" {
		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 5, tr(subTitle), "1", 1, "C", true, 0, "")
	}
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(70, 5, tr("RUTA"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 5, tr("FECHA DE VIAJE"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 5, tr(headerBoleto), "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 5, tr("N° PASE A BORDO"), "1", 1, "C", false, 0, "")

	if len(rows) == 0 {
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

func (s *ReportService) drawSegmentBlock(pdf *gofpdf.Fpdf, tr func(string) string, descargo *models.Descargo, title string, typeOrig, typeRepro models.TipoDetalleItinerario) {
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" "+title), "", 1, "L", false, 0, "")
	pdf.Ln(1)
	var origRows, reproRows []models.DetalleItinerarioDescargo
	for _, d := range descargo.DetallesItinerario {
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
	s.drawSubTable(pdf, tr, "", "N° BOLETO ORIGINAL", origRows)
	if len(reproRows) > 0 {
		pdf.Ln(1)
		s.drawSubTable(pdf, tr, "REPROGRAMACIÓN", "N° BOLETO REPROGRAMADO", reproRows)
	}
}

func (s *ReportService) drawReturnTableSummarized(pdf *gofpdf.Fpdf, tr func(string) string, subTitle string, boleto, ruta string, costo float64) {
	if subTitle != "" {
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 6, tr(subTitle), "", 1, "L", false, 0, "")
	}
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(90, 6, tr("RUTA COMPLETA"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 6, tr("N° BOLETO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 6, tr("COSTO TOTAL (Bs.)"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 8)
	if boleto != "" || ruta != "" {
		pdf.CellFormat(90, 8, tr(ruta), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, tr(boleto), "1", 0, "C", false, 0, "")
		costoStr := fmt.Sprintf("%.2f", costo)
		pdf.CellFormat(50, 8, tr(costoStr), "1", 1, "C", false, 0, "")
	} else {
		pdf.CellFormat(90, 8, "", "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, "", "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, "", "1", 1, "C", false, 0, "")
	}
}

func (s *ReportService) drawViaticosTable(pdf *gofpdf.Fpdf, tr func(string) string, viaticos []models.Viatico) {
	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("DESCARGO DE VIÁTICOS"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)
	for _, v := range viaticos {
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 5, tr("N° BOLETO / CÓDIGO VIÁTICO")+" : "+v.Codigo, "", 1, "L", false, 0, "")
		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(30, 5, tr("DESDE"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 5, tr("HASTA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(18, 5, tr("DÍAS"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(45, 5, tr("LUGAR"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(28, 5, tr("HABER/DÍA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(18, 5, "%", "1", 0, "C", true, 0, "")
		pdf.CellFormat(31, 5, tr("SUBTOTAL"), "1", 1, "C", true, 0, "")
		pdf.SetFont("Arial", "", 7)
		for _, d := range v.Detalles {
			pdf.CellFormat(30, 5, d.FechaDesde.Format("02/01/2006"), "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 5, d.FechaHasta.Format("02/01/2006"), "1", 0, "C", false, 0, "")
			pdf.CellFormat(18, 5, fmt.Sprintf("%.1f", d.Dias), "1", 0, "C", false, 0, "")
			pdf.CellFormat(45, 5, tr(d.Lugar), "1", 0, "L", false, 0, "")
			pdf.CellFormat(28, 5, fmt.Sprintf("%.2f", d.MontoDia), "1", 0, "R", false, 0, "")
			pdf.CellFormat(18, 5, fmt.Sprintf("%d", d.Porcentaje), "1", 0, "C", false, 0, "")
			pdf.CellFormat(31, 5, fmt.Sprintf("%.2f", d.SubTotal), "1", 1, "R", false, 0, "")
		}
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(169, 6, tr("TOTAL VIÁTICOS LÍQUIDO")+" :", "", 0, "R", false, 0, "")
		pdf.CellFormat(21, 6, fmt.Sprintf("%.2f", v.MontoLiquido), "1", 1, "R", false, 0, "")
		pdf.Ln(2)
		if v.TieneGastosRep && (v.MontoGastosRep > 0 || v.MontoLiquidoGastos > 0) {
			pdf.SetX(3)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(210, 6, tr("GASTOS DE REPRESENTACIÓN"), "B", 1, "L", false, 0, "")
			pdf.SetFont("Arial", "", 8)
			pdf.CellFormat(60, 5, tr("Monto asignado")+": "+fmt.Sprintf("%.2f", v.MontoGastosRep), "", 0, "L", false, 0, "")
			pdf.CellFormat(65, 5, tr("Retención")+": "+fmt.Sprintf("%.2f", v.MontoRetencionGastos), "", 0, "L", false, 0, "")
			pdf.CellFormat(65, 5, tr("Líquido")+": "+fmt.Sprintf("%.2f", v.MontoLiquidoGastos), "", 1, "L", false, 0, "")
			pdf.Ln(2)
		}
	}
	pdf.Ln(2)
}

func (s *ReportService) drawSignatureBlock(pdf *gofpdf.Fpdf, tr func(string) string, y float64, leftLabel, rightLabel, rightName string) {
	pdf.SetLineWidth(0.2)
	// Left side
	pdf.Line(35, y, 95, y)
	pdf.SetXY(35, y+2)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr(leftLabel), "", 1, "C", false, 0, "")

	// Right side
	pdf.Line(110, y, 185, y)
	pdf.SetXY(110, y+2)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(75, 4, tr(rightLabel), "", 1, "C", false, 0, "")
	if rightName != "" {
		pdf.SetX(110)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(75, 4, tr(rightName), "", 1, "C", false, 0, "")
	}
}

func (s *ReportService) GenerateViaticoV1(ctx context.Context, viatico *models.Viatico) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetX(3)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("(CALCULO DE VIATICOS) Art.13, parr. III, inc. 1 : Cuando el hospedaje sea cubierto por algun organismo financiador u otra entidad publica organizadora, las Senadoras, Senadores o Servidores Públicos, declarados en comisión al interior como exterior del país, percibirán solamente el 70% de los viaticos\nArt.13, parr. III, inc. 2 : Cuando el hospedaje y alimentación, sean cubiertos por algun organismo financiador u otra entidad publica organizadora, percibirán solamente el 25% de los viaticos")
		pdf.MultiCell(209, 2.8, disclaimer, "", "L", false)
		s.drawPageBorder(pdf)
	})

	pdf.AddPage()
	s.drawReportHeader(pdf, tr, "FORM-V-01", "FORMULARIO DE VIÁTICOS", "CÁMARA DE SENADORES", "GESTIÓN: "+viatico.CreatedAt.Format("2006"), viatico.Codigo)

	pdf.SetY(40)
	user := viatico.Usuario
	rolNombre := ""
	if user.Rol != nil {
		rolNombre = user.Rol.Nombre
	}
	s.drawLabelBox(pdf, tr, "A FAVOR DE :", user.GetNombreCompleto(), 40, 150, false)
	s.drawLabelBox(pdf, tr, "C.I. :", user.CI, 40, 60, true)
	s.drawLabelBox(pdf, tr, "CARGO/ROL :", rolNombre, 30, 60, false)

	unidadStr := "Senado Plurinacional"
	if user.Oficina != nil {
		unidadStr = user.Oficina.Detalle
	}
	s.drawLabelBox(pdf, tr, "UNIDAD :", unidadStr, 40, 150, false)

	pdf.Ln(2)

	if viatico.Solicitud != nil {
		sol := viatico.Solicitud
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(0, 8, tr("DATOS DE LA COMISIÓN"), "B", 1, "L", false, 0, "")
		pdf.Ln(2)

		fechaSol := "-"
		if fIda := sol.GetFechaIda(); fIda != nil {
			fechaSol = fIda.Format("02/01/2006")
		}
		s.drawLabelBox(pdf, tr, "SOLICITUD :", sol.Codigo, 40, 60, true)
		s.drawLabelBox(pdf, tr, "FECHA VIAJE :", fechaSol, 30, 60, false)
		s.drawLabelBox(pdf, tr, "MOTIVO :", sol.Motivo, 40, 150, false)
		s.drawLabelBox(pdf, tr, "LUGAR :", fmt.Sprintf("%s - %s", sol.GetOrigenCiudad(), sol.GetDestinoCiudad()), 40, 150, false)
	}

	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("DETALLE DE VIÁTICOS ASIGNADOS"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(30, 6, tr("Desde"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, tr("Hasta"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6, tr("Días"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 6, tr("Lugar"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 6, tr("Haber/Día (Bs)"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(15, 6, "%", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 6, tr("SubTotal (Bs)"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 8)
	for _, d := range viatico.Detalles {
		pdf.CellFormat(30, 6, d.FechaDesde.Format("02/01/2006"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, d.FechaHasta.Format("02/01/2006"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%.1f", d.Dias), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 6, tr(d.Lugar), "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", d.MontoDia), "1", 0, "R", false, 0, "")
		pdf.CellFormat(15, 6, fmt.Sprintf("%d %%", d.Porcentaje), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 6, fmt.Sprintf("%.2f", d.SubTotal), "1", 1, "R", false, 0, "")
	}

	pdf.Ln(2)
	xTotals := 130.0
	wL := 40.0
	wV := 30.0

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(wL, 6, tr("TOTAL VIATICO :"), "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(wV, 6, fmt.Sprintf("%.2f", viatico.MontoTotal), "1", 1, "R", false, 0, "")

	if viatico.TieneGastosRep {
		pdf.SetX(xTotals)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(wL, 6, tr("GASTOS REP. :"), "1", 0, "R", true, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(wV, 6, fmt.Sprintf("%.2f", viatico.MontoGastosRep), "1", 1, "R", false, 0, "")
	}

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(wL, 6, tr("RETENCION (13%) :"), "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "", 9)
	totalRet := viatico.MontoRC_IVA + viatico.MontoRetencionGastos
	pdf.CellFormat(wV, 6, fmt.Sprintf("%.2f", totalRet), "1", 1, "R", false, 0, "")

	pdf.SetX(xTotals)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(wL, 8, tr("LIQUIDO PAGABLE :"), "1", 0, "R", true, 0, "")
	pdf.SetFont("Arial", "B", 10)
	totalLiq := viatico.MontoLiquido + viatico.MontoLiquidoGastos
	pdf.CellFormat(wV, 8, fmt.Sprintf("%.2f", totalLiq), "1", 1, "R", false, 0, "")

	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Arial", "I", 9)
	literal := utils.NumeroALetras(totalLiq)
	pdf.CellFormat(190, 6, "Son: "+tr(literal)+" Bolivianos", "", 1, "L", false, 0, "")

	// Signatures
	sigY := pdf.GetY() + 15
	if sigY < 230 {
		sigY = 240
	}
	s.drawSignatureBlock(pdf, tr, sigY, "RECIBÍ CONFORME", "AUTORIZADO", viatico.Usuario.GetNombreCompleto())

	return pdf
}

func (s *ReportService) drawSolicitudSegment(ctx context.Context, pdf *gofpdf.Fpdf, tr func(string) string, title string, item *models.SolicitudItem, aerolineaSugerida string) {
	if item == nil {
		return
	}
	// Margen pequeño de 5mm para dar sensación de "borde a borde"
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
		pdf.CellFormat(w1, 8, item.UpdatedAt.Format("02/01/2006 15:04"), "1", 0, "C", false, 0, "")

		pdf.SetFont("Arial", "B", 7)
		pdf.SetTextColor(0, 0, 128)
		pdf.CellFormat(w2, 8, tr(item.GetEstado()), "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "", 7)

		aerolineaNombre := aerolineaSugerida
		if aerolinea, err := s.aerolineaRepo.FindByID(ctx, aerolineaSugerida); err == nil {
			if aerolinea.Sigla != "" {
				aerolineaNombre = aerolinea.Sigla
			} else {
				aerolineaNombre = aerolinea.Nombre
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

func (s *ReportService) drawPageBorder(pdf *gofpdf.Fpdf) {
	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Rect(3, 3, 210, 265.5, "D")
	pdf.Rect(3, 268.5, 210, 9.3, "D")
}
func (s *ReportService) GenerateConsolidadoPasajesExcel(ctx context.Context, filter dtos.ReportFilterRequest) (*excelize.File, error) {
	pasajes, err := s.pasajeRepo.FindConsolidado(ctx, filter)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Reporte de Pasajes"
	f.SetSheetName("Sheet1", sheet)

	// Estilos
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"03738C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	rowStyle, _ := f.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})

	// Encabezados
	headers := []string{"N°", "FECHA EMISIÓN", "CÓDIGO SOL.", "CONCEPTO", "BENEFICIARIO", "RUTA / TRAMOS", "AEROLÍNEA", "AGENCIA", "NRO. BOLETO", "COSTO (BS)", "ESTADO"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	totalCosto := 0.0
	for i, p := range pasajes {
		row := i + 2
		concepto := "DERECHO"
		codigoSolicitud := p.SolicitudID
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			concepto = p.SolicitudItem.Solicitud.GetConceptoCodigo()
			codigoSolicitud = p.SolicitudItem.Solicitud.Codigo
		}

		beneficiario := "-"
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			beneficiario = p.SolicitudItem.Solicitud.Usuario.GetNombreCompleto()
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		if p.FechaEmision != nil {
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.FechaEmision.Format("02/01/2006"))
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), codigoSolicitud)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), concepto)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), beneficiario)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), p.Ruta)
		if p.Aerolinea != nil {
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), p.Aerolinea.Nombre)
		}
		if p.Agencia != nil {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.Agencia.Nombre)
		}
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.NumeroBoleto)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), p.Costo)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), p.GetEstado())

		totalCosto += p.Costo

		// Aplicar estilo de borde a la fila
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), lastCell, rowStyle)
	}

	// Fila de Totales
	totalRow := len(pasajes) + 2
	f.SetCellValue(sheet, fmt.Sprintf("I%d", totalRow), "TOTAL GENERAL:")
	f.SetCellValue(sheet, fmt.Sprintf("J%d", totalRow), totalCosto)
	
	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F3F4F6"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("I%d", totalRow), fmt.Sprintf("J%d", totalRow), totalStyle)

	// Autoajustar anchos (aproximado)
	f.SetColWidth(sheet, "A", "A", 5)
	f.SetColWidth(sheet, "B", "B", 15)
	f.SetColWidth(sheet, "C", "C", 15)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 35)
	f.SetColWidth(sheet, "F", "F", 40)
	f.SetColWidth(sheet, "G", "G", 15)
	f.SetColWidth(sheet, "H", "H", 15)
	f.SetColWidth(sheet, "I", "I", 20)
	f.SetColWidth(sheet, "J", "J", 15)
	f.SetColWidth(sheet, "K", "K", 12)

	return f, nil
}
