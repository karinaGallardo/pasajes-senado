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

	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/jung-kurt/gofpdf"
	"github.com/xuri/excelize/v2"
)

type ReportService struct {
	solicitudRepo  *repositories.SolicitudRepository
	aerolineaRepo  *repositories.AerolineaRepository
	pasajeRepo     *repositories.PasajeRepository
	agenciaRepo    *repositories.AgenciaRepository
	cupoRepo       *repositories.CupoDerechoRepository
	openTicketRepo *repositories.OpenTicketRepository
	configService  *ConfiguracionService
}

func NewReportService(
	solicitudRepo *repositories.SolicitudRepository,
	aerolineaRepo *repositories.AerolineaRepository,
	pasajeRepo *repositories.PasajeRepository,
	agenciaRepo *repositories.AgenciaRepository,
	cupoRepo *repositories.CupoDerechoRepository,
	openTicketRepo *repositories.OpenTicketRepository,
	configService *ConfiguracionService,
) *ReportService {
	return &ReportService{
		solicitudRepo:  solicitudRepo,
		aerolineaRepo:  aerolineaRepo,
		pasajeRepo:     pasajeRepo,
		agenciaRepo:    agenciaRepo,
		cupoRepo:       cupoRepo,
		openTicketRepo: openTicketRepo,
		configService:  configService,
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

func (s *ReportService) GeneratePV05(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) (*gofpdf.Fpdf, error) {
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

	drawSubTable := func(subTitle string, headerBillete string, rows []models.DescargoTramo) {
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
		pdf.CellFormat(35, 5, tr("ORIGEN"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 5, tr("DESTINO"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(30, 5, tr("FECHA DE VIAJE"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 5, tr(headerBillete), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 5, tr("N° PASE A BORDO"), "1", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 8)
		for _, r := range rows {
			orig := r.GetRutaOrigen()
			dest := r.GetRutaDestino()

			if r.Origen != nil {
				orig = r.Origen.GetNombreCorto()
			}
			if r.Destino != nil {
				dest = r.Destino.GetNombreCorto()
			}

			pdf.CellFormat(35, 6, tr(orig), "1", 0, "L", false, 0, "")
			pdf.CellFormat(35, 6, tr(dest), "1", 0, "L", false, 0, "")
			fecha := ""
			if r.Fecha != nil {
				fecha = r.Fecha.Format("02/01/2006")
			}
			pdf.CellFormat(30, 6, tr(fecha), "1", 0, "C", false, 0, "")
			pdf.CellFormat(50, 6, tr(r.Billete), "1", 0, "C", false, 0, "")

			paseVal := r.NumeroPaseAbordo
			if r.EsOpenTicket {
				pdf.SetTextColor(200, 0, 0)
				pdf.SetFont("Arial", "B", 7)
				paseVal = "TRAMO NO UTILIZADO"
			} else if r.EsModificacion {
				pdf.SetTextColor(0, 128, 0) // Green
				pdf.SetFont("Arial", "B", 7)
				paseVal = "MODIFICADO"
			}
			pdf.CellFormat(40, 6, tr(paseVal), "1", 1, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 8)
		}
	}

	// Group by Ticket consolidate devo/mod status
	type ItinerarioReporte struct {
		Billete        string
		Tramos         []models.DescargoTramo
		EsOpenTicket   bool
		EsModificacion bool
	}
	itinerariosMap := make(map[string]*ItinerarioReporte)
	var itinerariosOrder []string

	for _, d := range descargo.Tramos {
		key := d.Billete
		if key == "" {
			key = "SN-" + d.GetRutaDisplay()
		}
		if _, ok := itinerariosMap[key]; !ok {
			itinerariosMap[key] = &ItinerarioReporte{Billete: d.Billete}
			itinerariosOrder = append(itinerariosOrder, key)
		}
		if d.EsOpenTicket {
			itinerariosMap[key].EsOpenTicket = true
		}
		if d.EsModificacion {
			itinerariosMap[key].EsModificacion = true
		}
		itinerariosMap[key].Tramos = append(itinerariosMap[key].Tramos, d)
	}

	drawSegmentBlock := func(title string, typeOrig, typeRepro models.TipoDescargoTramo, tMap map[string]*ItinerarioReporte, tOrder []string) {
		var origRows, reproRows []models.DescargoTramo

		for _, key := range tOrder {
			g := tMap[key]
			for i := range g.Tramos {
				d := &g.Tramos[i]
				switch d.Tipo {
				case typeOrig:
					origRows = append(origRows, *d)
				case typeRepro:
					reproRows = append(reproRows, *d)
				}
			}
		}

		if len(origRows) > 0 || len(reproRows) > 0 {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(190, 6, tr(" "+title), "", 1, "L", false, 0, "")
			pdf.Ln(1)
			drawSubTable("", "N° BILLETE ORIGINAL", origRows)
			if len(reproRows) > 0 {
				pdf.Ln(1)
				drawSubTable("REPROGRAMACIÓN", "N° BILLETE REPROGRAMADO", reproRows)
			}
		}
	}

	drawSegmentBlock("TRAMO DE IDA", models.TipoTramoIdaOriginal, models.TipoTramoIdaReprogramada, itinerariosMap, itinerariosOrder)
	pdf.Ln(4)
	drawSegmentBlock("TRAMO DE RETORNO", models.TipoTramoVueltaOriginal, models.TipoTramoVueltaReprogramada, itinerariosMap, itinerariosOrder)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" TRAMOS NO UTILIZADOS (OPEN TICKETS REUTILIZABLES)"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	hasReturns := false
	for _, key := range itinerariosOrder {
		g := itinerariosMap[key]
		if g.EsOpenTicket {
			hasReturns = true
			// Reconstruir ruta solo de los tramos marcados como open
			var routeParts []string
			for _, det := range g.Tramos {
				if det.EsOpenTicket {
					routeParts = append(routeParts, det.GetRutaDisplay())
				}
			}
			fullRoute := strings.Join(routeParts, " ; ")
			s.drawReturnTableSummarized(pdf, tr, "TRAMO NO UTILIZADO", g.Billete, fullRoute)
			pdf.Ln(1)
		}
	}

	if !hasReturns {
		// Si no hay Open Tickets, mostramos una fila indicando que no hubo tramos sobrantes
		s.drawReturnTableSummarized(pdf, tr, "N/A", "NINGUNO", "TODOS LOS TRAMOS FUERON UTILIZADOS")
	}
	pdf.Ln(4)

	// --- SECCIÓN: LIQUIDACIÓN FINANCIERA DETALLADA ---
	pdf.SetFont("Arial", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(190, 7, tr(" LIQUIDACIÓN FINANCIERA Y CONCILIACIÓN DE COSTOS (Bs)"), "1", 1, "C", true, 0, "")

	// Calcular totales desglosados
	totalEmitido := 0.0
	totalUtilizado := 0.0
	totalEfectivo := 0.0
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				totalEmitido += p.Costo
				totalUtilizado += p.CostoUtilizado
				totalEfectivo += p.MontoReembolso
			}
		}
	}

	pdf.SetFont("Arial", "B", 7)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(25, 6, tr("N° BILLETE"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 6, tr("DETALLE RUTA"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 6, tr("EMITIDO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 6, tr("CONSUMO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 6, tr("DEVOLUCIÓN"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 6, tr("N° BOLETA"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 7)
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				pdf.CellFormat(25, 6, tr(p.NumeroBillete), "1", 0, "C", false, 0, "")
				pdf.CellFormat(70, 6, tr(p.GetRutaDisplay()), "1", 0, "L", false, 0, "")
				pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", p.Costo), "1", 0, "R", false, 0, "")
				pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", p.CostoUtilizado), "1", 0, "R", false, 0, "")

				if p.MontoReembolso > 0 {
					pdf.SetTextColor(150, 0, 0)
					pdf.SetFont("Arial", "B", 7)
				}
				pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", p.MontoReembolso), "1", 0, "R", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 7)

				nroBoleta := p.NroBoletaDeposito
				if nroBoleta == "" {
					nroBoleta = "-"
				}
				pdf.CellFormat(20, 6, tr(nroBoleta), "1", 1, "C", false, 0, "")
			}
		}
	}

	// Fila de Totales
	pdf.SetFillColor(245, 245, 245)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(95, 6, tr("TOTALES GENERALES "), "1", 0, "R", true, 0, "")
	pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", totalEmitido), "1", 0, "R", true, 0, "")
	pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", totalUtilizado), "1", 0, "R", true, 0, "")

	pdf.SetTextColor(150, 0, 0)
	pdf.CellFormat(25, 6, fmt.Sprintf("%.2f", totalEfectivo), "1", 0, "R", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(20, 6, "", "1", 1, "C", true, 0, "")

	pdf.Ln(10)
	sigY := pdf.GetY() + 15
	if sigY > 258 {
		pdf.AddPage()
		sigY = 30
	}
	s.drawSignatureBlock(pdf, tr, sigY, "SELLO UNIDAD SOLICITANTE", "", "", "FIRMA/RESPONSABLE PRESENTACION DEL DESCARGO", "", "")

	// --- ANEXO AUTOMÁTICO DEL COMPROBANTE DE DEPÓSITO ---
	// Se coloca al final
	compPath := ""
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				if p.ArchivoComprobante != "" {
					compPath = p.ArchivoComprobante
					break
				}
			}
			if compPath != "" {
				break
			}
		}
	}

	if descargo.GetTotalDevolucionPasajes() > 0 && compPath != "" {
		validPath, isTemp, err := s.getValidImage(compPath)
		if err == nil {
			if isTemp {
				defer os.Remove(validPath)
			}
			pdf.AddPage()
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(190, 8, tr("ANEXO: COMPROBANTE DE DEPÓSITO BANCARIO"), "B", 1, "C", false, 0, "")
			pdf.Ln(5)

			lowerPath := strings.ToLower(validPath)
			var opt gofpdf.ImageOptions
			if strings.HasSuffix(lowerPath, ".jfif") {
				opt.ImageType = "JPG"
			}
			pdf.ImageOptions(validPath, 10, 30, 190, 0, false, opt, 0, "")
		}
	}

	if pdf.Err() {
		return nil, fmt.Errorf("error en generación de PDF base PV-05: %v", pdf.Error())
	}
	return pdf, nil
}

func (s *ReportService) GeneratePV06(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) (*gofpdf.Fpdf, error) {
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
	s.drawReportHeader(pdf, tr, "FORM-PV-06", "INFORME DE DESCARGO DE VIAJE", "PASAJES OFICIALES", "SERVIDORES PÚBLICOS - GESTION: "+solicitud.CreatedAt.Format("2006"), solicitud.Codigo)

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
	s.drawMemoRow(pdf, tr, "Fecha :", descargo.FechaPresentacion.Format("02/01/2006"))

	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("Datos Generales del Viaje:"), "", 1, "L", false, 0, "")

	// Row for "Lugar del Viaje"
	pdf.SetFillColor(255, 255, 255) // Peach color from image
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(35, 8, tr("Lugar del Viaje:"), "1", 0, "L", true, 0, "")
	pdf.SetFont("Arial", "", 10)
	lugarViaje := "-"
	if descargo.Oficial != nil && descargo.Oficial.LugarViaje != "" {
		lugarViaje = descargo.Oficial.LugarViaje
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

	// Datos de Salida y Retorno (PV-06)
	if descargo.Oficial != nil {
		pdf.SetX(10) // Reset X to standard margin
		pdf.Ln(4)
		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 9)

		// Encabezados
		pdf.CellFormat(95, 6, tr("DATOS DE SALIDA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(95, 6, tr("DATOS DE RETORNO"), "1", 1, "C", true, 0, "")

		pdf.SetFont("Arial", "", 9)

		fSalida := "-"
		hSalida := "-"
		if !descargo.Oficial.FechaSalida.IsZero() {
			fSalida = descargo.Oficial.FechaSalida.Format("02/01/2006")
			hSalida = descargo.Oficial.FechaSalida.Format("15:04")
		}

		fRetorno := "-"
		hRetorno := "-"
		if !descargo.Oficial.FechaRetorno.IsZero() {
			fRetorno = descargo.Oficial.FechaRetorno.Format("02/01/2006")
			hRetorno = descargo.Oficial.FechaRetorno.Format("15:04")
		}

		// Fila de Fechas
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(15, 6, tr("  Fecha:"), "L", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(80, 6, tr(fSalida), "R", 0, "L", false, 0, "")

		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(15, 6, tr("  Fecha:"), "L", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(80, 6, tr(fRetorno), "R", 1, "L", false, 0, "")

		// Fila de Horas
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(15, 6, tr("  Hora:"), "LB", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(80, 6, tr(hSalida), "RB", 0, "L", false, 0, "")

		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(15, 6, tr("  Hora:"), "LB", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(80, 6, tr(hRetorno), "RB", 1, "L", false, 0, "")
	}

	pdf.Ln(4)

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

	sectionIdx := 1
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(40, 6, tr(fmt.Sprintf("%d. OBJETIVO DEL VIAJE:", sectionIdx)), "", 0, "L", false, 0, "")
	sectionIdx++
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(150, 6, tr(objViaje), "", "L", false)
	pdf.Ln(4)

	// Actividades
	infActividades := "Sin información"
	if descargo.Oficial != nil && descargo.Oficial.InformeActividades != "" {
		infActividades = descargo.Oficial.InformeActividades
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(fmt.Sprintf("%d. ACTIVIDADES REALIZADAS:", sectionIdx)), "", 1, "L", false, 0, "")
	sectionIdx++
	pdf.SetFont("Arial", "", 9)
	pdf.SetX(15) // Sangría
	pdf.MultiCell(175, 5, tr(infActividades), "", "J", false)
	pdf.Ln(4)

	// Resultados
	if descargo.Oficial != nil && descargo.Oficial.ResultadosViaje != "" {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(fmt.Sprintf("%d. RESULTADOS OBTENIDOS:", sectionIdx)), "", 1, "L", false, 0, "")
		sectionIdx++
		pdf.SetFont("Arial", "", 9)
		pdf.SetX(15)
		pdf.MultiCell(175, 5, tr(descargo.Oficial.ResultadosViaje), "", "J", false)
		pdf.Ln(4)
	}

	// Conclusiones
	if descargo.Oficial != nil && descargo.Oficial.ConclusionesRecomendaciones != "" {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(fmt.Sprintf("%d. CONCLUSIONES:", sectionIdx)), "", 1, "L", false, 0, "")
		sectionIdx++
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
		pdf.CellFormat(190, 8, tr(fmt.Sprintf("%d. Anexos (Reseña fotográfica)", sectionIdx)), "B", 1, "C", false, 0, "")
		sectionIdx++
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
				// Validar y obtener una ruta de imagen que gofpdf entienda
				validPath, isTemp, err := s.getValidImage(anexo.Archivo)
				if err != nil {
					continue
				}
				if isTemp {
					defer os.Remove(validPath)
				}

				lowerPath := strings.ToLower(validPath)
				imgType := ""
				if strings.HasSuffix(lowerPath, ".jfif") {
					imgType = "JPG"
				}

				pdf.RegisterImage(validPath, imgType)
				if pdf.Err() {
					pdf.ClearError()
					continue
				}
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
				pdf.CellFormat(0, 6, tr(fmt.Sprintf("%d. Anexos (Reseña fotográfica) (Continuación)", sectionIdx-1)), "", 1, "R", false, 0, "")
				pdf.Ln(4)
			}

			// Draw the row
			startX := marginSide + (pageWidth-(h*totalAspect+float64(len(rowIndices)-1)*imgSpacing))/2.0
			currX := startX
			currY := pdf.GetY()
			for _, idx := range rowIndices {
				validPath, isTemp, _ := s.getValidImage(anexos[idx].Archivo)
				if isTemp {
					defer os.Remove(validPath)
				}

				info := pdf.GetImageInfo(validPath)
				asp := info.Width() / info.Height()
				w := h * asp
				pdf.Image(validPath, currX, currY, w, 0, false, "", 0, "")
				currX += w + imgSpacing
			}

			pdf.SetY(currY + h + rowSpacing)
			i += len(rowIndices)
		}
	}

	// --- Nueva Página para Tramos e Itinerario ---
	pdf.AddPage()

	// Transporte y Devoluciones
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(50, 6, tr(fmt.Sprintf("%d. TRANSPORTE UTILIZADO:", sectionIdx)), "", 1, "L", false, 0, "")
	sectionIdx++

	if descargo.Oficial != nil {
		pdf.SetFont("Arial", "", 9)

		// Caso Aéreo
		if descargo.Oficial.HasTransportType("AEREO") {
			pdf.SetX(15)
			pdf.CellFormat(140, 6, tr("- Transporte Aéreo"), "", 1, "L", false, 0, "")
		}

		// Caso Terrestre Público
		if descargo.Oficial.HasTransportType("TERRESTRE_PUBLICO") {
			pdf.SetX(15)
			pdf.CellFormat(140, 6, tr("- Transporte Terrestre Público"), "", 1, "L", false, 0, "")
			if len(descargo.Oficial.TransportesTerrestres) > 0 {
				pdf.Ln(1)
				pdf.SetX(15)
				pdf.SetFillColor(240, 240, 240)
				pdf.SetFont("Arial", "B", 7)
				pdf.CellFormat(30, 6, tr("FECHA"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(30, 6, tr("SENTIDO"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(50, 6, tr("N° FACTURA"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(40, 6, tr("IMPORTE (Bs.)"), "1", 1, "C", true, 0, "")

				pdf.SetFont("Arial", "", 8)
				for _, tt := range descargo.Oficial.TransportesTerrestres {
					pdf.SetX(15)
					pdf.CellFormat(30, 6, tt.Fecha.Format("02/01/2006"), "1", 0, "C", false, 0, "")
					pdf.CellFormat(30, 6, tr(tt.Tipo), "1", 0, "C", false, 0, "")
					pdf.CellFormat(50, 6, tr(tt.NroFactura), "1", 0, "C", false, 0, "")
					pdf.CellFormat(40, 6, fmt.Sprintf("%.2f", tt.Importe), "1", 1, "R", false, 0, "")
				}
				pdf.Ln(2)
			}
		}

		// Caso Vehículo Oficial
		if descargo.Oficial.HasTransportType("VEHICULO_OFICIAL") {
			pdf.SetX(15)
			pdf.SetFont("Arial", "", 9)
			msg := "- Uso de Vehículo Oficial"
			if descargo.Oficial.PlacaVehiculo != "" {
				msg += fmt.Sprintf(" (N° de Placa: %s)", descargo.Oficial.PlacaVehiculo)
			}
			pdf.CellFormat(140, 6, tr(msg), "", 1, "L", false, 0, "")
		}
	} else {
		pdf.SetX(15)
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(140, 6, tr("No especificado"), "", 1, "L", false, 0, "")
	}

	pdf.Ln(6)

	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("DESCARGO DE PASAJES (ADJUNTAR PASES A BORDO)"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	s.drawSegmentBlock(pdf, tr, descargo, "TRAMO DE IDA", models.TipoTramoIdaOriginal, models.TipoTramoIdaReprogramada)
	pdf.Ln(4)
	s.drawSegmentBlock(pdf, tr, descargo, "TRAMO DE RETORNO", models.TipoTramoVueltaOriginal, models.TipoTramoVueltaReprogramada)
	pdf.Ln(4)

	if len(solicitud.Viaticos) > 0 {
		s.drawViaticosTable(pdf, tr, solicitud.Viaticos)
	}

	pdf.Ln(4)

	// --- SECCIÓN DE LIQUIDACIÓN FINANCIERA DETALLADA ---
	if descargo.GetTotalDevolucionPasajes() >= 0 { // Mostrar siempre para conciliación
		pdf.SetX(3)
		pdf.SetFont("Arial", "B", 10)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(210, 8, tr(fmt.Sprintf(" %d. LIQUIDACIÓN FINANCIERA (CONCILIACIÓN DE COSTOS DE PASAJES)", sectionIdx)), "1", 1, "L", true, 0, "")
		sectionIdx++

		totalEmitido := 0.0
		totalUtilizado := 0.0
		totalEfectivo := 0.0
		if descargo.Solicitud != nil {
			for _, item := range descargo.Solicitud.Items {
				for _, p := range item.Pasajes {
					totalEmitido += p.Costo
					totalUtilizado += p.CostoUtilizado
					totalEfectivo += p.MontoReembolso
				}
			}
		}

		// Header de la tabla
		pdf.SetX(3)
		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(30, 7, tr("N° BILLETE"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(75, 7, tr("DETALLE RUTA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, tr("EMITIDO (Bs.)"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, tr("CONSUMO (Bs.)"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, tr("DEVOLUCIÓN (Bs.)"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, tr("N° BOLETA"), "1", 1, "C", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		if descargo.Solicitud != nil {
			for _, item := range descargo.Solicitud.Items {
				for _, p := range item.Pasajes {
					pdf.SetX(3)
					pdf.CellFormat(30, 7, tr(p.NumeroBillete), "1", 0, "C", false, 0, "")
					pdf.CellFormat(75, 7, tr(p.GetRutaDisplay()), "1", 0, "L", false, 0, "")
					pdf.CellFormat(25, 7, fmt.Sprintf("%.2f", p.Costo), "1", 0, "R", false, 0, "")
					pdf.CellFormat(25, 7, fmt.Sprintf("%.2f", p.CostoUtilizado), "1", 0, "R", false, 0, "")

					if p.MontoReembolso > 0 {
						pdf.SetTextColor(150, 0, 0)
						pdf.SetFont("Arial", "B", 8)
					}
					pdf.CellFormat(25, 7, fmt.Sprintf("%.2f", p.MontoReembolso), "1", 0, "R", false, 0, "")
					pdf.SetTextColor(0, 0, 0)
					pdf.SetFont("Arial", "", 8)

					nroBoleta := p.NroBoletaDeposito
					if nroBoleta == "" {
						nroBoleta = "-"
					}
					pdf.CellFormat(30, 7, tr(nroBoleta), "1", 1, "C", false, 0, "")
				}
			}
		}

		// Totales
		pdf.SetX(3)
		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(105, 8, tr("TOTALES GENERALES (Bs.) "), "1", 0, "R", true, 0, "")
		pdf.CellFormat(25, 8, fmt.Sprintf("%.2f", totalEmitido), "1", 0, "R", true, 0, "")
		pdf.CellFormat(25, 8, fmt.Sprintf("%.2f", totalUtilizado), "1", 0, "R", true, 0, "")

		pdf.SetTextColor(150, 0, 0)
		pdf.CellFormat(25, 8, fmt.Sprintf("%.2f", totalEfectivo), "1", 0, "R", true, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(30, 8, "", "1", 1, "C", true, 0, "")
		pdf.Ln(4)
	}

	pdf.Ln(4)
	pdf.SetX(10)
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(190, 10, tr("Es cuanto se informa para fines consiguientes."), "", 1, "L", false, 0, "")

	pdf.Ln(2)

	// --- Signatures (Dynamic Position) ---
	sigY := pdf.GetY() + 15

	// Ensure we don't start signatures too low on the page
	if sigY > 259 {
		pdf.AddPage()
		sigY = 30
	}

	pdf.SetY(sigY)

	cargo := ""
	if solicitud.Usuario.Cargo != nil {
		cargo = solicitud.Usuario.Cargo.Descripcion
	}

	if solicitud.Usuario.IsSenador() {
		s.drawSignatureBlock(pdf, tr, sigY+10, "SELLO UNIDAD SOLICITANTE", "", "", "FIRMA Y SELLO SENADOR(A)", solicitud.Usuario.GetNombreCompleto(), cargo)
	} else {
		s.drawSignatureBlock(pdf, tr, sigY+10, "FIRMA Y SELLO SERVIDOR PÚBLICO", solicitud.Usuario.GetNombreCompleto(), cargo, "Vo.Bo. Inmediato Superior", "", "")
	}

	if pdf.Err() {
		return nil, fmt.Errorf("error en generación de PDF base PV-06: %v", pdf.Error())
	}
	return pdf, nil
}

func (s *ReportService) GeneratePV05Complete(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) ([]byte, error) {
	// 1. Generar el PDF Base PV-05
	pdf, err := s.GeneratePV05(ctx, descargo, personaView)
	if err != nil {
		return nil, err
	}

	// Crear archivos temporales para la unión
	tmpBase, err := os.CreateTemp("", "pv05_base_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBase.Name())

	if err := pdf.OutputFileAndClose(tmpBase.Name()); err != nil {
		return nil, fmt.Errorf("error generating base PDF: %w", err)
	}

	// 2. Recolectar rutas de archivos que existan
	var filesToMerge []string
	filesToMerge = append(filesToMerge, tmpBase.Name())

	// Mapa para evitar duplicados
	seenFiles := make(map[string]bool)
	seenFiles[tmpBase.Name()] = true

	// 2.1 Billetes Electrónicos (Pasajes emitidos) y Facturas de Servicio
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, pasaje := range item.Pasajes {
				if pasaje.Archivo != "" && !seenFiles[pasaje.Archivo] {
					if s.isValidPDF(pasaje.Archivo) {
						filesToMerge = append(filesToMerge, pasaje.Archivo)
						seenFiles[pasaje.Archivo] = true
					}
				}
				if pasaje.ServicioArchivo != "" && !seenFiles[pasaje.ServicioArchivo] {
					if s.isValidPDF(pasaje.ServicioArchivo) {
						filesToMerge = append(filesToMerge, pasaje.ServicioArchivo)
						seenFiles[pasaje.ServicioArchivo] = true
					}
				}
			}
		}
	}

	// 2.2 Pases a Bordo (Solo de tramos que NO son de reutilizaci�n y NO son Open Ticket)
	for _, det := range descargo.Tramos {
		// En el descargo normal, solo queremos los pases de lo que se vol� originalmente
		if !det.IsReutilizacion() && !det.EsOpenTicket && det.ArchivoPaseAbordo != "" && !seenFiles[det.ArchivoPaseAbordo] {
			if s.isValidPDF(det.ArchivoPaseAbordo) {
				filesToMerge = append(filesToMerge, det.ArchivoPaseAbordo)
				seenFiles[det.ArchivoPaseAbordo] = true
			}
		}
	}

	// 2.3 Comprobante de Depósito (Si es PDF se une)
	compPath := ""
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				if p.ArchivoComprobante != "" {
					compPath = p.ArchivoComprobante
					break
				}
			}
			if compPath != "" {
				break
			}
		}
	}

	if descargo.GetTotalDevolucionPasajes() > 0 && compPath != "" {
		if strings.HasSuffix(strings.ToLower(compPath), ".pdf") && !seenFiles[compPath] {
			if s.isValidPDF(compPath) {
				filesToMerge = append(filesToMerge, compPath)
				seenFiles[compPath] = true
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

func (s *ReportService) GeneratePV05OpenTicket(ctx context.Context, descargo *models.Descargo) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		s.drawPageBorder(pdf)
		pdf.SetFont("Arial", "I", 7)
		pdf.CellFormat(0, 10, tr(fmt.Sprintf("Página %d", pdf.PageNo())), "", 0, "C", false, 0, "")
	})

	pdf.AddPage()

	gestion := descargo.CreatedAt.Format("2006")
	s.drawReportHeader(pdf, tr, "FORM-PV-05", "REPORTE DE BILLETES PARA REUTILIZACIÓN", "CÁMARA DE SENADORES", "GESTIÓN: "+gestion, descargo.Codigo)

	pdf.SetY(40)
	user := descargo.Solicitud.Usuario
	s.drawLabelBox(pdf, tr, "BENEFICIARIO :", user.GetNombreCompleto(), 40, 150, false)
	s.drawLabelBox(pdf, tr, "C.I. :", user.CI, 40, 60, true)

	cargo := "-"
	if user.Cargo != nil {
		cargo = user.Cargo.Descripcion
	}
	s.drawLabelBox(pdf, tr, "CARGO :", cargo, 30, 60, false)

	// 1. Agrupamiento por Pasaje (Contenedor del crédito)
	type PasajeGroup struct {
		Pasaje     *models.Pasaje
		Originales []models.DescargoTramo
		Nuevos     []models.DescargoTramo
	}

	pasajesMap := make(map[string]*PasajeGroup)
	var ordenPasajes []string // Para mantener un orden determinista

	for _, t := range descargo.Tramos {
		if t.PasajeID == nil {
			continue
		}
		pid := *t.PasajeID

		if _, ok := pasajesMap[pid]; !ok {
			pasajesMap[pid] = &PasajeGroup{Pasaje: t.Pasaje}
			ordenPasajes = append(ordenPasajes, pid)
		}

		if t.EsOpenTicket {
			pasajesMap[pid].Originales = append(pasajesMap[pid].Originales, t)
		} else if t.IsReutilizacion() {
			pasajesMap[pid].Nuevos = append(pasajesMap[pid].Nuevos, t)
		}
	}

	// SECCIÓN 1: Detalle de Reutilización de Billetes
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("1. DETALLE DE REUTILIZACIÓN DE BILLETES (MOVIMIENTOS)"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	hasAnyOT := false
	for _, pid := range ordenPasajes {
		group := pasajesMap[pid]
		if len(group.Originales) == 0 {
			continue
		}
		hasAnyOT = true

		// CABECERA DEL PASAJE (Con indicación de Ida/Vuelta)
		tipoPasaje := ""
		if len(group.Originales) > 0 {
			if strings.HasPrefix(string(group.Originales[0].Tipo), "VUELTA") {
				tipoPasaje = " [VUELTA]"
			} else {
				tipoPasaje = " [IDA]"
			}
		}

		pdf.SetFillColor(230, 235, 245)
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 8, tr("BILLETE ORIGINAL N° "+group.Pasaje.NumeroBillete+tipoPasaje), "1", 1, "L", true, 0, "")

		// Sub-encabezado de Tramos
		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(80, 6, tr("RUTA / TRAMO"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(40, 6, tr("N° BILLETE"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 6, tr("FECHA"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(40, 6, tr("ESTADO / TIPO"), "1", 1, "C", true, 0, "")

		// A. LISTAR TRAMOS ORIGINALES (NO UTILIZADOS)
		pdf.SetFont("Arial", "", 8)
		for _, t := range group.Originales {
			fecha := "-"
			if t.Fecha != nil {
				fecha = t.Fecha.Format("02/01/2006")
			}

			pdf.CellFormat(80, 7, tr(t.GetRutaDisplay()), "1", 0, "L", false, 0, "")
			pdf.CellFormat(40, 7, tr(t.Billete), "1", 0, "C", false, 0, "")
			pdf.CellFormat(30, 7, tr(fecha), "1", 0, "C", false, 0, "")

			pdf.SetTextColor(200, 0, 0)
			pdf.SetFont("Arial", "B", 7)
			pdf.CellFormat(40, 7, tr("NO UTILIZADO"), "1", 1, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Arial", "", 8)
		}

		// B. LISTAR REUTILIZACIONES (NUEVOS VUELOS)
		if len(group.Nuevos) > 0 {
			for _, n := range group.Nuevos {
				fechaN := "-"
				if n.Fecha != nil {
					fechaN = n.Fecha.Format("02/01/2006")
				}

				pdf.SetFillColor(250, 255, 250)
				pdf.CellFormat(80, 7, tr(n.GetRutaDisplay()), "1", 0, "L", true, 0, "")

				billeteDisplay := n.Billete
				if n.Billete != group.Pasaje.NumeroBillete {
					pdf.SetFont("Arial", "", 7)
					billeteDisplay = fmt.Sprintf("%s", n.Billete)
				}
				pdf.CellFormat(40, 7, tr(billeteDisplay), "1", 0, "C", true, 0, "")
				pdf.SetFont("Arial", "", 8)

				pdf.CellFormat(30, 7, tr(fechaN), "1", 0, "C", true, 0, "")

				pdf.SetTextColor(0, 100, 0)
				pdf.SetFont("Arial", "B", 7)
				pdf.CellFormat(40, 7, tr("REUTILIZADO"), "1", 1, "C", true, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 8)
			}
		}
		pdf.Ln(3)
	}

	if !hasAnyOT {
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(190, 8, tr("No se han registrado movimientos de reutilización en este descargo."), "1", 1, "C", false, 0, "")
	}

	// SECCIÓN 3: Liquidación Financiera y Conciliación de Costos
	pdf.Ln(5)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 8, tr("3. LIQUIDACIÓN FINANCIERA Y CONCILIACIÓN DE COSTOS (Bs)"), "B", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(25, 7, tr("N° BILLETE"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 7, tr("DETALLE RUTA"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 7, tr("EMITIDO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 7, tr("CONSUMO"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 7, tr("DEVOLUCIÓN"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(29, 7, tr("N° BOLETA"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 7)
	hasFinances := false
	var totalEmitido, totalConsumo, totalDevolucion float64

	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				// Mostramos todos los pasajes para conciliación, no solo los que tienen reembolso
				hasFinances = true

				emitido := p.Costo
				consumo := p.CostoUtilizado
				if consumo == 0 && p.MontoReembolso == 0 {
					consumo = emitido // Si no se especificó consumo, asumimos consumo total
				}
				devolucion := p.MontoReembolso

				totalEmitido += emitido
				totalConsumo += consumo
				totalDevolucion += devolucion

				pdf.CellFormat(25, 7, tr(p.NumeroBillete), "1", 0, "C", false, 0, "")
				pdf.CellFormat(70, 7, tr(p.GetRutaDisplay()), "1", 0, "L", false, 0, "")
				pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", emitido), "1", 0, "R", false, 0, "")
				pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", consumo), "1", 0, "R", false, 0, "")

				if devolucion > 0 {
					pdf.SetTextColor(150, 0, 0)
					pdf.SetFont("Arial", "B", 7)
				}
				pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", devolucion), "1", 0, "R", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 7)

				nro := p.NroBoletaDeposito
				if nro == "" {
					nro = "-"
				}
				pdf.CellFormat(29, 7, tr(nro), "1", 1, "C", false, 0, "")
			}
		}
	}

	if hasFinances {
		// Fila de Totales
		pdf.SetFillColor(245, 245, 245)
		pdf.SetFont("Arial", "B", 7)
		pdf.CellFormat(95, 7, tr("TOTALES GENERALES"), "1", 0, "R", true, 0, "")
		pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", totalEmitido), "1", 0, "R", true, 0, "")
		pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", totalConsumo), "1", 0, "R", true, 0, "")
		pdf.CellFormat(22, 7, fmt.Sprintf("%.2f", totalDevolucion), "1", 0, "R", true, 0, "")
		pdf.CellFormat(29, 7, "", "1", 1, "C", true, 0, "")
	} else {
		pdf.CellFormat(190, 8, tr("No se registraron movimientos financieros en este descargo."), "1", 1, "C", false, 0, "")
	}

	pdf.Ln(10)
	pdf.SetFont("Arial", "B", 9)
	pdf.MultiCell(190, 5, tr("Se certifica que los tramos detallados corresponden a pasajes previamente devueltos y no utilizados, que fueron pagados en su emisión original y posteriormente reutilizados y utilizados por el beneficiario, conforme a normativa vigente."), "", "L", false)

	// Firmas (Igual que PV-05 estándar)
	sigY := pdf.GetY() + 30
	if sigY > 240 {
		pdf.AddPage()
		sigY = 40
	}
	s.drawSignatureBlock(pdf, tr, sigY, "SELLO UNIDAD SOLICITANTE", "", "", "FIRMA/RESPONSABLE PRESENTACION DEL DESCARGO", "", "")

	return pdf
}

func (s *ReportService) GeneratePV05OpenTicketComplete(ctx context.Context, descargo *models.Descargo) ([]byte, error) {
	// 1. Generar el PDF Base PV-05 OT
	pdf := s.GeneratePV05OpenTicket(ctx, descargo)
	if pdf.Err() {
		return nil, pdf.Error()
	}

	// Crear archivos temporales para la unión
	tmpBase, err := os.CreateTemp("", "pv05_ot_base_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBase.Name())

	if err := pdf.OutputFileAndClose(tmpBase.Name()); err != nil {
		return nil, fmt.Errorf("error generating base OT PDF: %w", err)
	}

	// 2. Recolectar rutas de archivos que existan
	var filesToMerge []string
	filesToMerge = append(filesToMerge, tmpBase.Name())

	// Mapa para evitar duplicados
	seenFiles := make(map[string]bool)
	seenFiles[tmpBase.Name()] = true

	// 2.1 Billetes Electrónicos de los tramos marcados como Open Ticket
	for _, tramo := range descargo.Tramos {
		if tramo.EsOpenTicket {
			// Buscar el pasaje correspondiente en la solicitud para obtener el archivo del billete
			if descargo.Solicitud != nil {
				for _, item := range descargo.Solicitud.Items {
					for _, pasaje := range item.Pasajes {
						// Si el número de billete coincide, intentamos adjuntarlo
						if pasaje.NumeroBillete == tramo.Billete && pasaje.Archivo != "" && !seenFiles[pasaje.Archivo] {
							if s.isValidPDF(pasaje.Archivo) {
								filesToMerge = append(filesToMerge, pasaje.Archivo)
								seenFiles[pasaje.Archivo] = true
							}
						}
					}
				}
			}
		}
	}

	// 2.2 Pases a Bordo de los tramos de REUTILIZACIÓN
	for _, tramo := range descargo.Tramos {
		// En el reporte OT, solo nos interesan los pases de los nuevos vuelos (REUT)
		if tramo.IsReutilizacion() && tramo.ArchivoPaseAbordo != "" && !seenFiles[tramo.ArchivoPaseAbordo] {
			if s.isValidPDF(tramo.ArchivoPaseAbordo) {
				filesToMerge = append(filesToMerge, tramo.ArchivoPaseAbordo)
				seenFiles[tramo.ArchivoPaseAbordo] = true
			}
		}
	}

	// 2.2 Comprobantes de Depósito (Devoluciones)
	compPath := ""
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				if p.ArchivoComprobante != "" {
					compPath = p.ArchivoComprobante
					if !seenFiles[compPath] && s.isValidPDF(compPath) {
						filesToMerge = append(filesToMerge, compPath)
						seenFiles[compPath] = true
					}
				}
			}
		}
	}

	// 3. Si solo hay un archivo (el base), retornarlo directamente
	if len(filesToMerge) == 1 {
		data, err := os.ReadFile(tmpBase.Name())
		if err != nil {
			return nil, fmt.Errorf("error reading base OT PDF: %w", err)
		}
		return data, nil
	}

	// 4. Unir usando pdftk
	tmpFinal, err := os.CreateTemp("", "pv05_ot_final_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFinal.Name())

	args := append(filesToMerge, "cat", "output", tmpFinal.Name())
	cmd := exec.CommandContext(ctx, "pdftk", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("error merging OT PDFs with pdftk: %w - %s", err, stderr.String())
	}

	data, err := os.ReadFile(tmpFinal.Name())
	if err != nil {
		return nil, fmt.Errorf("error reading final merged OT PDF: %w", err)
	}
	return data, nil
}

func (s *ReportService) GeneratePV06Complete(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) ([]byte, error) {
	// 1. Generar el PDF Base PV-06
	pdf, err := s.GeneratePV06(ctx, descargo, personaView)
	if err != nil {
		return nil, err
	}

	// Crear archivos temporales para la unión
	tmpBase, err := os.CreateTemp("", "pv06_base_*.pdf")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpBase.Name())

	if err := pdf.OutputFileAndClose(tmpBase.Name()); err != nil {
		return nil, fmt.Errorf("error saving base PDF: %w", err)
	}

	// 2. Recolectar rutas de archivos que existan (Billetes Electrónicos + Pases a Bordo)
	var filesToMerge []string
	filesToMerge = append(filesToMerge, tmpBase.Name())

	// Mapa para evitar duplicados (ej: mismo billete para ida y vuelta)
	seenFiles := make(map[string]bool)
	seenFiles[tmpBase.Name()] = true

	// 2.0 Memorandum PDF (If uploaded)
	if descargo.Oficial != nil && descargo.Oficial.ArchivoMemorandum != "" {
		path := descargo.Oficial.ArchivoMemorandum
		if !seenFiles[path] {
			if s.isValidPDF(path) {
				filesToMerge = append(filesToMerge, path)
				seenFiles[path] = true
			}
		}
	}

	// 2.1 Billetes Electrónicos (Pasajes emitidos) y Facturas de Servicio
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, pasaje := range item.Pasajes {
				if pasaje.Archivo != "" && !seenFiles[pasaje.Archivo] {
					if s.isValidPDF(pasaje.Archivo) {
						filesToMerge = append(filesToMerge, pasaje.Archivo)
						seenFiles[pasaje.Archivo] = true
					}
				}
				if pasaje.ServicioArchivo != "" && !seenFiles[pasaje.ServicioArchivo] {
					if s.isValidPDF(pasaje.ServicioArchivo) {
						filesToMerge = append(filesToMerge, pasaje.ServicioArchivo)
						seenFiles[pasaje.ServicioArchivo] = true
					}
				}
			}
		}
	}

	// 2.2 Pases a Bordo (Cargados en el descargo)
	for _, det := range descargo.Tramos {
		if det.ArchivoPaseAbordo != "" && !seenFiles[det.ArchivoPaseAbordo] {
			if s.isValidPDF(det.ArchivoPaseAbordo) {
				filesToMerge = append(filesToMerge, det.ArchivoPaseAbordo)
				seenFiles[det.ArchivoPaseAbordo] = true
			}
		}
	}

	// 2.3 Comprobante de Depósito (Si es PDF se une)
	compPath := ""
	if descargo.Solicitud != nil {
		for _, item := range descargo.Solicitud.Items {
			for _, p := range item.Pasajes {
				if p.ArchivoComprobante != "" {
					compPath = p.ArchivoComprobante
					break
				}
			}
			if compPath != "" {
				break
			}
		}
	}

	if descargo.GetTotalDevolucionPasajes() > 0 && compPath != "" {
		if strings.HasSuffix(strings.ToLower(compPath), ".pdf") && !seenFiles[compPath] {
			if s.isValidPDF(compPath) {
				filesToMerge = append(filesToMerge, compPath)
				seenFiles[compPath] = true
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

func (s *ReportService) drawSubTable(pdf *gofpdf.Fpdf, tr func(string) string, subTitle string, headerBillete string, rows []models.DescargoTramo) {
	if subTitle != "" {
		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 5, tr(subTitle), "1", 1, "C", true, 0, "")
	}
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(35, 5, tr("ORIGEN"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(35, 5, tr("DESTINO"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(30, 5, tr("FECHA DE VIAJE"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(50, 5, tr(headerBillete), "1", 0, "C", false, 0, "")
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
		parts := strings.Split(r.GetRutaDisplay(), " - ")
		if len(parts) >= 2 {
			orig = strings.TrimSpace(parts[0])
			dest = strings.TrimSpace(parts[1])
		} else {
			orig = r.GetRutaDisplay()
		}
		pdf.CellFormat(35, 6, tr(orig), "1", 0, "C", false, 0, "")
		pdf.CellFormat(35, 6, tr(dest), "1", 0, "C", false, 0, "")
		fecha := ""
		if r.Fecha != nil {
			fecha = r.Fecha.Format("02/01/2006")
		}
		pdf.CellFormat(30, 6, tr(fecha), "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 6, tr(r.Billete), "1", 0, "C", false, 0, "")
		paseVal := r.NumeroPaseAbordo
		if r.EsOpenTicket {
			pdf.SetTextColor(200, 0, 0)
			pdf.SetFont("Arial", "B", 7)
			paseVal = "TRAMO NO UTILIZADO"
		} else if r.EsModificacion {
			pdf.SetTextColor(0, 128, 0) // Green
			pdf.SetFont("Arial", "B", 7)
			paseVal = "MODIFICADO"
		}
		pdf.CellFormat(40, 6, tr(paseVal), "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Arial", "", 8)
	}
}

func (s *ReportService) drawSegmentBlock(pdf *gofpdf.Fpdf, tr func(string) string, descargo *models.Descargo, title string, typeOrig, typeRepro models.TipoDescargoTramo) {
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" "+title), "", 1, "L", false, 0, "")
	pdf.Ln(1)
	var origRows, reproRows []models.DescargoTramo
	for _, d := range descargo.Tramos {
		if d.EsOpenTicket {
			continue
		}
		switch d.Tipo {
		case typeOrig:
			origRows = append(origRows, d)
		case typeRepro:
			reproRows = append(reproRows, d)
		}
	}
	s.drawSubTable(pdf, tr, "", "N° BILLETE ORIGINAL", origRows)
	if len(reproRows) > 0 {
		pdf.Ln(1)
		s.drawSubTable(pdf, tr, "REPROGRAMACIÓN", "N° BILLETE REPROGRAMADO", reproRows)
	}
}

func (s *ReportService) drawReturnTableSummarized(pdf *gofpdf.Fpdf, tr func(string) string, subTitle string, billete, ruta string) {
	if subTitle != "" {
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 6, tr(subTitle), "", 1, "L", false, 0, "")
	}
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Arial", "B", 8)
	pdf.CellFormat(120, 6, tr("TRAMO / RUTA"), "1", 0, "C", true, 0, "")
	pdf.CellFormat(70, 6, tr("N° BILLETE"), "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 8)
	if billete != "" || ruta != "" {
		pdf.CellFormat(120, 8, tr(ruta), "1", 0, "C", false, 0, "")
		pdf.CellFormat(70, 8, tr(billete), "1", 1, "C", false, 0, "")
	} else {
		pdf.CellFormat(120, 8, "", "1", 0, "C", false, 0, "")
		pdf.CellFormat(70, 8, "", "1", 1, "C", false, 0, "")
	}
}

func (s *ReportService) drawViaticosTable(pdf *gofpdf.Fpdf, tr func(string) string, viaticos []models.Viatico) {
	pdf.SetX(3)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(210, 8, tr("DESCARGO DE VIÁTICOS"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)
	for _, v := range viaticos {
		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(190, 5, tr("N° BILLETE / CÓDIGO VIÁTICO")+" : "+v.Codigo, "", 1, "L", false, 0, "")
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

func (s *ReportService) drawSignatureBlock(pdf *gofpdf.Fpdf, tr func(string) string, y float64, leftLabel, leftName, leftTitle, rightLabel, rightName, rightTitle string) {
	pdf.SetLineWidth(0.2)
	// Left side
	pdf.Line(35, y, 95, y)
	pdf.SetXY(35, y+2)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr(leftLabel), "", 1, "C", false, 0, "")
	if leftName != "" {
		pdf.SetX(35)
		pdf.SetFont("Arial", "", 7)
		pdf.CellFormat(60, 4, tr(leftName), "", 1, "C", false, 0, "")
	}
	if leftTitle != "" {
		pdf.SetX(35)
		pdf.SetFont("Arial", "I", 6)
		pdf.CellFormat(60, 3, tr(leftTitle), "", 1, "C", false, 0, "")
	}

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
	if rightTitle != "" {
		pdf.SetX(110)
		pdf.SetFont("Arial", "I", 6)
		pdf.CellFormat(75, 3, tr(rightTitle), "", 1, "C", false, 0, "")
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
	s.drawSignatureBlock(pdf, tr, sigY, "RECIBÍ CONFORME", "", "", "AUTORIZADO", viatico.Usuario.GetNombreCompleto(), "")

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
		pdf.CellFormat(w1, 8, item.UpdatedAt.Format("02/01/2006 15:04"), "1", 0, "C", false, 0, "")

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

func (s *ReportService) drawPageBorder(pdf *gofpdf.Fpdf) {
	pdf.SetLineWidth(0.3)
	pdf.SetDrawColor(0, 0, 0)
	pdf.Rect(3, 3, 210, 265.5, "D")
	pdf.Rect(3, 268.5, 210, 9.3, "D")
}

// isValidPDF verifica si un archivo existe y tiene la cabecera mágica de un PDF (%PDF-)
func (s *ReportService) isValidPDF(filePath string) bool {
	if filePath == "" {
		return false
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	header := make([]byte, 5)
	n, err := f.Read(header)
	if err != nil || n < 5 {
		return false
	}

	return string(header) == "%PDF-"
}

// getValidImage verifica si un archivo es una imagen y lo convierte a PNG si el formato no es soportado por gofpdf (ej: webp, bmp)
func (s *ReportService) getValidImage(filePath string) (string, bool, error) {
	if filePath == "" {
		return "", false, fmt.Errorf("empty path")
	}

	f, err := os.Open(filePath)
	if err != nil {
		return "", false, err
	}
	defer f.Close()

	// Detectar formato
	img, format, err := image.Decode(f)
	if err != nil {
		return "", false, fmt.Errorf("decoding error: %w", err)
	}

	// Formatos soportados nativamente por gofpdf
	if format == "jpeg" || format == "png" || format == "gif" {
		return filePath, false, nil
	}

	// Si es otro formato (webp, bmp, etc.), convertir a PNG temporal
	tmpFile, err := os.CreateTemp("", "img_conv_*.png")
	if err != nil {
		return "", false, err
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		return "", false, err
	}

	return tmpFile.Name(), true, nil
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
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"03738C"}, Pattern: 1},
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
	headers := []string{
		"N°", "FECHA EMISIÓN", "CÓDIGO SOL.", "CONCEPTO", "TIPO DE SOLICITUD", "BENEFICIARIO", "RUTA / TRAMOS", "AEROLÍNEA", "AGENCIA", "NRO. BILLETE",
		"COSTO ORIGEN (BS)", "DEV DIF TARIFA", "COSTO CONSUMO", "CARGOS ASOCIADOS", "COSTO TOTAL",
		"ESTADO PASAJE", "ESTADO SOLICITUD", "ESTADO DESCARGO",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	totalCostoOrigen := 0.0
	totalCargos := 0.0
	totalGeneral := 0.0

	for i, p := range pasajes {
		row := i + 2
		concepto := "DERECHO"
		tipoSolicitud := "-"
		codigoSolicitud := p.SolicitudID
		solicitudEstado := "-"
		descargoEstado := "NO PRESENTADO"

		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			sol := p.SolicitudItem.Solicitud
			concepto = sol.GetConceptoCodigo()
			codigoSolicitud = sol.Codigo

			if concepto == "DERECHO" {
				tipoSolicitud = "POR DERECHO"
			} else {
				tipoSolicitud = strings.ToUpper(sol.GetTipoNombre())
			}

			if sol.EstadoSolicitud != nil {
				solicitudEstado = strings.ToUpper(sol.EstadoSolicitud.Nombre)
			}

			if sol.Descargo != nil {
				descargoEstado = strings.ToUpper(sol.Descargo.GetEstadoLabel())
			}
		}

		beneficiario := "-"
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			beneficiario = p.SolicitudItem.Solicitud.Usuario.GetNombreCompleto()
		}

		montoCargos := p.GetMontoCargos()
		costoTotalPasaje := p.Costo + montoCargos

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		if p.FechaEmision != nil {
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.FechaEmision.Format("02/01/2006"))
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), codigoSolicitud)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), concepto)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), tipoSolicitud)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), beneficiario)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), p.GetRutaDisplay())

		if p.Aerolinea != nil {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), p.Aerolinea.Sigla)
		}
		if p.Agencia != nil {
			f.SetCellValue(sheet, fmt.Sprintf("I%d", row), p.Agencia.Nombre)
		}

		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), p.NumeroBillete)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), p.Costo)
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), p.MontoReembolso)
		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), p.CostoUtilizado)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), montoCargos)
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), costoTotalPasaje)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), p.GetEstado())
		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), solicitudEstado)
		f.SetCellValue(sheet, fmt.Sprintf("R%d", row), descargoEstado)

		totalCostoOrigen += p.Costo
		totalCargos += montoCargos
		totalGeneral += costoTotalPasaje

		// Aplicar estilo de borde a la fila
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), lastCell, rowStyle)
	}

	// Fila de Totales
	totalRow := len(pasajes) + 2
	f.SetCellValue(sheet, fmt.Sprintf("J%d", totalRow), "TOTALES:")
	f.SetCellValue(sheet, fmt.Sprintf("K%d", totalRow), totalCostoOrigen)
	f.SetCellValue(sheet, fmt.Sprintf("N%d", totalRow), totalCargos)
	f.SetCellValue(sheet, fmt.Sprintf("O%d", totalRow), totalGeneral)

	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F3F4F6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("J%d", totalRow), fmt.Sprintf("O%d", totalRow), totalStyle)

	// Autoajustar anchos (aproximado)
	widths := []float64{5, 15, 15, 12, 20, 35, 45, 12, 25, 15, 18, 15, 15, 18, 18, 15, 18, 18}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	return f, nil
}

func (s *ReportService) GenerateMorosidadDescargosExcel(ctx context.Context) (*excelize.File, error) {
	solicitudes, err := s.solicitudRepo.FindPendientesDeDescargo(ctx)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Morosidad de Descargos"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"893026"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headers := []string{"NOMBRE COMPLETO", "CÓDIGO SOLICITUD", "MOTIVO / CONCEPTO", "ÚLTIMO VUELO", "DÍAS DE MORA"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	rowIndex := 2
	for _, sol := range solicitudes {
		diasMora := sol.GetDiasRestantesDescargo()
		if diasMora >= 0 {
			continue // Solo morosos
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), sol.Usuario.GetNombreCompleto())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), sol.Codigo)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), sol.GetConceptoNombre())
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIndex), sol.GetUltimoVueloFecha())
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIndex), -diasMora)
		rowIndex++
	}

	f.SetColWidth(sheet, "A", "A", 40)
	f.SetColWidth(sheet, "B", "B", 20)
	f.SetColWidth(sheet, "C", "C", 30)
	f.SetColWidth(sheet, "D", "D", 15)
	f.SetColWidth(sheet, "E", "E", 15)

	return f, nil
}

func (s *ReportService) GenerateUsoCuposExcel(ctx context.Context, anio, mes int) (*excelize.File, error) {
	cupos, err := s.cupoRepo.FindByPeriodo(ctx, anio, mes)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Uso de Cupos"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2A3B56"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headers := []string{"SENADOR(A)", "GESTIÓN", "MES", "LÍMITE MENSUAL", "USADOS", "DISPONIBLES"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for i, c := range cupos {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), c.SenTitular.GetNombreCompleto())
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), c.Gestion)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), utils.TranslateMonth(time.Month(c.Mes)))
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), c.CupoTotal)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), c.CupoUsado)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), c.CupoTotal-c.CupoUsado)
	}

	f.SetColWidth(sheet, "A", "A", 40)
	f.SetColWidth(sheet, "B", "F", 15)

	return f, nil
}

func (s *ReportService) GenerateEstadisticasAerolineaExcel(ctx context.Context, filter dtos.ReportFilterRequest) (*excelize.File, error) {
	pasajes, err := s.pasajeRepo.FindConsolidado(ctx, filter)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]struct {
		Count int
		Total float64
	})

	for _, p := range pasajes {
		name := "OTRA / DESCONOCIDA"
		if p.Aerolinea != nil {
			name = p.Aerolinea.Nombre
		}
		curr := stats[name]
		curr.Count++
		curr.Total += p.Costo
		stats[name] = curr
	}

	f := excelize.NewFile()
	sheet := "Estadísticas por Aerolínea"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"0F7654"}, Pattern: 1},
	})

	headers := []string{"AEROLÍNEA", "CANTIDAD PASAJES", "INVERSIÓN TOTAL (BS)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	rowIndex := 2
	for name, data := range stats {
		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIndex), name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIndex), data.Count)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIndex), data.Total)
		rowIndex++
	}

	f.SetColWidth(sheet, "A", "A", 30)
	f.SetColWidth(sheet, "B", "C", 20)

	return f, nil
}

func (s *ReportService) GenerateOficialesExcel(ctx context.Context, filter dtos.ReportFilterRequest) (*excelize.File, error) {
	solicitudes, err := s.solicitudRepo.FindOficialesForReport(ctx, filter)
	if err != nil {
		return nil, err
	}

	f := excelize.NewFile()
	sheet := "Reporte Pasajes Oficiales"
	f.SetSheetName("Sheet1", sheet)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"03738C"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	rowStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
		},
	})

	headers := []string{
		"CÓDIGO SOLICITUD", "NOMBRE Y APELLIDOS", "CARGO", "OFICINA", "NRO. MEMO/NOTA", "ITINERARIO / RUTA",
		"F. SALIDA", "F. RETORNO", "F. LÍMITE DESCARGO", "MONTO TOTAL (BS)", "F. DESCARGO",
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}

	for i, sol := range solicitudes {
		row := i + 2

		cargo := "-"
		if sol.Usuario.Cargo != nil {
			cargo = sol.Usuario.Cargo.Descripcion
		}

		oficina := "-"
		if sol.Usuario.Oficina != nil {
			oficina = sol.Usuario.Oficina.Detalle
		}

		itinerario := sol.GetItinerarioResumen()

		var fSalida, fRetorno time.Time
		for _, item := range sol.Items {
			if item.Fecha != nil {
				if fSalida.IsZero() || item.Fecha.Before(fSalida) {
					fSalida = *item.Fecha
				}
				if fRetorno.IsZero() || item.Fecha.After(fRetorno) {
					fRetorno = *item.Fecha
				}
			}
		}

		fLimite := ""
		if !fRetorno.IsZero() {
			fLimite = utils.CalcularFechaLimiteDescargo(fRetorno).Format("02/01/2006")
		}

		montoTotal := 0.0
		for _, item := range sol.Items {
			for _, p := range item.Pasajes {
				montoTotal += p.Costo
			}
		}

		fDescargo := "-"
		if sol.Descargo != nil && !sol.Descargo.CreatedAt.IsZero() {
			fDescargo = sol.Descargo.CreatedAt.Format("02/01/2006")
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), sol.Codigo)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), sol.Usuario.GetNombreCompleto())
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), cargo)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), oficina)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), sol.Autorizacion)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), itinerario)

		if !fSalida.IsZero() {
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), fSalida.Format("02/01/2006"))
		}
		if !fRetorno.IsZero() {
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), fRetorno.Format("02/01/2006"))
		}

		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), fLimite)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), montoTotal)
		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), fDescargo)

		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), lastCell, rowStyle)
	}

	widths := []float64{18, 30, 25, 25, 20, 40, 15, 15, 18, 18, 15}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, w)
	}

	return f, nil
}
