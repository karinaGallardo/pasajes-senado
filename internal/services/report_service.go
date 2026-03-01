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

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("Las Senadoras y Senadores, deberan solicitar Pasajes POR DERECHO mediante el formulario respectivo a ser presentado en el área de Pasajes con 48 horas previas al viaje consignando la información correspondiente. Asi mismo, deberán presentar los pases a bordo originales de ida y vuelta del ultimo viaje efectuado mediante formulario respectivo y en el plazo de 8 días.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
	})

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
	drawLabelBox("MES DE VIAJE :", mesYNum, 40, 50, false)

	pdf.Ln(5)

	// SEGMENTS LOGIC
	drawSegment := func(title string, item *models.SolicitudItem) {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(190, 6, tr(" "+title), "B", 1, "L", false, 0, "")

		if item != nil && item.GetEstado() != "PENDIENTE" {
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
			pdf.CellFormat(190, 10, tr(" TRAMO NO SOLICITADO"), "", 1, "C", false, 0, "")
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
	pdf.CellFormat(40, 6, tr("Nro(Memo/RD/Nota JG/RC) :"), "", 0, "R", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(150, 6, "  "+tr(authVal), "1", 1, "L", false, 0, "")

	pdf.Ln(2)

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

	return pdf
}

// GeneratePV02 genera el formulario FORM-PV-02 para solicitud de pasajes oficial (funcionarios).
// personaView puede ser nil; si viene de MongoDB (por CI) se usan Cargo y Dependencia para el PDF.
func (s *ReportService) GeneratePV02(ctx context.Context, solicitud *models.Solicitud, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("NO SE DARA CURSO AL TRAMITE DE PASAJES Y VIATICOS, EN CASO DE VERIFICARSE QUE LOS DOCUMENTOS DE DECLARATORIA Y/O DESCARGO EN COMISION OFICIAL PRESENTEN ALTERACIONES, BORRONES O ENMIENDAS QUE MODIFIQUEN EL CONTENIDO DE LA MISMA, SIN PERJUICIO DE INICIAR LAS ACCIONES LEGALES QUE CORRESPONDA.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
	})

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	hHeader := 22.0

	pdf.SetLineWidth(0.1)
	pdf.Line(xHeader, yHeader+hHeader, xHeader+167, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+3)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-02", "", 1, "C", false, 0, "")

	pdf.SetX(xHeader)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(40, 8, fmt.Sprintf("%s", solicitud.Codigo), "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE SOLICITUD"), "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES AEREOS PARA FUNCIONARIOS(AS)"), "", 1, "C", false, 0, "")
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

	// DATOS DEL SUSCRITO
	drawLabelBox("NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 45, 145, false)
	drawLabelBox("C.I. :", solicitud.Usuario.CI, 45, 55, true)
	drawLabelBox("TEL. REF :", solicitud.Usuario.Phone, 35, 55, false)

	cargoStr := ""
	if personaView != nil && personaView.Cargo != "" {
		cargoStr = personaView.Cargo
	} else if solicitud.Usuario.Cargo != nil {
		cargoStr = solicitud.Usuario.Cargo.Descripcion
	}
	drawLabelBox("CARGO :", cargoStr, 45, 145, false)

	unidadStr := ""
	if personaView != nil && personaView.Dependencia != "" {
		unidadStr = personaView.Dependencia
	} else if solicitud.Usuario.Oficina != nil {
		unidadStr = solicitud.Usuario.Oficina.Detalle
	}
	drawLabelBox("UNIDAD FUNCIONAL :", unidadStr, 45, 145, false)

	fechaSol := solicitud.CreatedAt.Format("02/01/2006")
	horaSol := solicitud.CreatedAt.Format("15:04")
	drawLabelBox("FECHA DE SOLICITUD :", fechaSol, 45, 55, true)
	drawLabelBox("HORA :", horaSol, 25, 40, false)

	authVal := solicitud.Autorizacion
	if authVal == "" {
		authVal = "PD"
	}
	drawLabelBox("Nro(Memo/RD/Nota JG/RC) :", authVal, 45, 55, false)

	drawLabelBox("CONCEPTO DE VIAJE :", "OFICIAL", 45, 55, false)

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
	} else if aerolinea, err := s.aerolineaRepo.FindByID(aerolineaNombre); err == nil {
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
			if aerolinea, err := s.aerolineaRepo.FindByID(solicitud.AerolineaSugerida); err == nil {
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
			} else if item.Hora != "" {
				hora = item.Hora
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

	// Firmas
	pdf.SetFont("Arial", "B", 7)
	pdf.Line(25, 248, 95, 248)
	pdf.SetXY(25, 250)
	pdf.CellFormat(70, 4, tr("SELLO UNIDAD SOLICITANTE"), "", 1, "C", false, 0, "")

	pdf.Line(115, 248, 175, 248)
	pdf.SetXY(115, 250)
	pdf.CellFormat(60, 4, tr("FIRMA / SELLO SOLICITANTE"), "", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "I", 8)
	return pdf
}

func (s *ReportService) GeneratePV05(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	solicitud := descargo.Solicitud

	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("NO SE DARA CURSO AL TRAMITE DE PASAJES Y VIATICOS, EN CASO DE VERIFICARSE QUE LOS DOCUMENTOS DE DECLARATORIA Y/O DESCARGO EN COMISION OFICIAL PRESENTEN ALTERACIONES, BORRONES O ENMIENDAS QUE MODIFIQUEN EL CONTENIDO DE LA MISMA, SIN PERJUICIO DE INICIAR LAS ACCIONES LEGALES QUE CORRESPONDA.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
	})

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
	pdf.CellFormat(40, 8, fmt.Sprintf("%s", solicitud.Codigo), "", 1, "C", false, 0, "")
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

	drawLabelBox("NOMBRE Y APELLIDOS :", solicitud.Usuario.GetNombreCompleto(), 40, 150, false)
	drawLabelBox("C.I. :", solicitud.Usuario.CI, 40, 60, true)
	drawLabelBox("TEL. REF :", solicitud.Usuario.Phone, 30, 60, false)

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

	drawSubTable := func(subTitle string, headerBoleto string, rows []models.DetalleItinerarioDescargo) {
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

	drawSegmentBlock := func(title string, typeOrig, typeRepro models.TipoDetalleItinerario) {
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
		drawSubTable("", "N° BOLETO ORIGINAL", origRows)
		if len(reproRows) > 0 {
			pdf.Ln(1)
			drawSubTable("REPROGRAMACIÓN", "N° BOLETO REPROGRAMADO", reproRows)
		}
	}

	drawSegmentBlock("TRAMO DE IDA", models.TipoDetalleIdaOriginal, models.TipoDetalleIdaReprogramada)
	pdf.Ln(4)
	drawSegmentBlock("TRAMO DE RETORNO", models.TipoDetalleVueltaOriginal, models.TipoDetalleVueltaReprogramada)
	pdf.Ln(4)

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" PASAJE ABIERTO-OPEN TICKET"), "B", 1, "L", false, 0, "")

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

	sigY := 220.0
	pdf.SetY(sigY)
	pdf.SetFont("Arial", "B", 8)
	pdf.Line(35, sigY+10, 95, sigY+10)
	pdf.SetXY(35, sigY+12)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr("SENADORA / SENADOR"), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "I", 6)
	pdf.CellFormat(60, 3, tr("(FIRMA Y SELLO)"), "", 1, "C", false, 0, "")

	pdf.Line(110, sigY+10, 180, sigY+10)
	pdf.SetXY(110, sigY+12)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(70, 4, tr("RESPONSABLE PRESENTACION DEL DESCARGO"), "", 1, "C", false, 0, "")

	return pdf
}

func (s *ReportService) GeneratePV06(ctx context.Context, descargo *models.Descargo, personaView *models.MongoPersonaView) *gofpdf.Fpdf {
	solicitud := descargo.Solicitud

	pdf := gofpdf.New("P", "mm", "Letter", "")
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("Arial", "I", 6)
		pdf.SetTextColor(50, 50, 50)
		disclaimer := tr("NO SE DARA CURSO AL TRAMITE DE PASAJES Y VIATICOS, EN CASO DE VERIFICARSE QUE LOS DOCUMENTOS DE DECLARATORIA Y/O DESCARGO EN COMISION OFICIAL PRESENTEN ALTERACIONES, BORRONES O ENMIENDAS QUE MODIFIQUEN EL CONTENIDO DE LA MISMA, SIN PERJUICIO DE INICIAR LAS ACCIONES LEGALES QUE CORRESPONDA.")
		pdf.MultiCell(190, 3, disclaimer, "", "L", false)
	})

	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	xHeader, yHeader := 10.0, 10.0
	hHeader := 22.0

	pdf.SetLineWidth(0.1)
	pdf.Line(xHeader, yHeader+hHeader, xHeader+167, yHeader+hHeader)

	pdf.SetXY(xHeader, yHeader+3)
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(40, 6, "FORM-PV-06", "", 1, "C", false, 0, "")

	pdf.SetX(xHeader)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(40, 8, fmt.Sprintf("%s", solicitud.Codigo), "", 1, "C", false, 0, "")
	pdf.SetXY(xHeader+40, yHeader+4)
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(115, 8, tr("FORMULARIO DE DESCARGO"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+11)
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(115, 6, tr("PASAJES OFICIALES"), "", 1, "C", false, 0, "")

	pdf.SetXY(xHeader+40, yHeader+16)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(115, 6, tr("SERVIDORES PÚBLICOS")+" - "+fmt.Sprintf("GESTION: %s", solicitud.CreatedAt.Format("2006")), "", 1, "C", false, 0, "")

	pdf.Image("web/static/img/logo_senado.png", xHeader+160, yHeader-6, 30, 0, false, "", 0, "")

	pdf.SetY(40)
	pdf.SetFont("Arial", "", 10)

	cargoStr := ""
	if personaView != nil && personaView.Cargo != "" {
		cargoStr = personaView.Cargo
	} else if solicitud.Usuario.Cargo != nil {
		cargoStr = solicitud.Usuario.Cargo.Descripcion
	}

	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)

	drawMemoRow := func(label, value string) {
		h := 7.0
		pdf.SetFillColor(255, 255, 255) // White background
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(30, h, tr(label), "1", 0, "R", true, 0, "")

		pdf.SetFillColor(255, 255, 255)
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(160, h, "  "+tr(value), "1", 1, "L", false, 0, "")
	}

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

	drawMemoRow("A :", aQuien)
	drawMemoRow("De :", solicitud.Usuario.GetNombreCompleto())
	drawMemoRow("Cargo :", cargoStr)
	drawMemoRow("Fecha :", solicitud.CreatedAt.Format("02/01/2006"))

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

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 8, tr("DESCARGO DE PASAJES (ADJUNTAR PASES A BORDO)"), "B", 1, "C", false, 0, "")
	pdf.Ln(2)

	drawSubTable := func(subTitle string, headerBoleto string, rows []models.DetalleItinerarioDescargo) {
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

	drawSegmentBlock := func(title string, typeOrig, typeRepro models.TipoDetalleItinerario) {
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
		drawSubTable("", "N° BOLETO ORIGINAL", origRows)
		if len(reproRows) > 0 {
			pdf.Ln(1)
			drawSubTable("REPROGRAMACIÓN", "N° BOLETO REPROGRAMADO", reproRows)
		}
	}

	drawSegmentBlock("TRAMO DE IDA", models.TipoDetalleIdaOriginal, models.TipoDetalleIdaReprogramada)
	pdf.Ln(4)
	drawSegmentBlock("TRAMO DE RETORNO", models.TipoDetalleVueltaOriginal, models.TipoDetalleVueltaReprogramada)
	pdf.Ln(4)

	if len(solicitud.Viaticos) > 0 {
		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(190, 8, tr("DESCARGO DE VIÁTICOS"), "B", 1, "C", false, 0, "")
		pdf.Ln(2)
		for _, v := range solicitud.Viaticos {
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
				pdf.SetFont("Arial", "B", 9)
				pdf.CellFormat(190, 6, tr("GASTOS DE REPRESENTACIÓN"), "B", 1, "L", false, 0, "")
				pdf.SetFont("Arial", "", 8)
				pdf.CellFormat(60, 5, tr("Monto asignado")+": "+fmt.Sprintf("%.2f", v.MontoGastosRep), "", 0, "L", false, 0, "")
				pdf.CellFormat(65, 5, tr("Retención")+": "+fmt.Sprintf("%.2f", v.MontoRetencionGastos), "", 0, "L", false, 0, "")
				pdf.CellFormat(65, 5, tr("Líquido")+": "+fmt.Sprintf("%.2f", v.MontoLiquidoGastos), "", 1, "L", false, 0, "")
				pdf.Ln(2)
			}
		}
		pdf.Ln(2)
	}

	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(190, 6, tr(" DEVOLUCIÓN DE PASAJES (si aplica)"), "B", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(190, 5, tr(" (En caso de no haber utilizado el boleto emitido o un tramo informar en el siguiente cuadro)"), "", 1, "L", false, 0, "")

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
	pdf.Ln(5)

	// INFORME DETALLADO PV-06
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(190, 8, tr("INFORME DE COMISIÓN"), "B", 1, "C", false, 0, "")
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
	pdf.CellFormat(40, 6, tr("N° MEMORÁNDUM:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.MultiCell(150, 6, tr(nroMemo), "", "L", false)
	pdf.Ln(1)

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

	// Transporte y Devolución
	tipoTrans := "No especificado"
	if descargo.Oficial != nil && descargo.Oficial.TipoTransporte != "" {
		tipoTrans = descargo.Oficial.TipoTransporte
		if descargo.Oficial.PlacaVehiculo != "" {
			tipoTrans += " (Placa: " + descargo.Oficial.PlacaVehiculo + ")"
		}
	}
	pdf.SetFont("Arial", "B", 9)
	pdf.CellFormat(50, 6, tr("TRANSPORTE UTILIZADO:"), "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(140, 6, tr(tipoTrans), "", 1, "L", false, 0, "")

	if descargo.Oficial != nil && descargo.Oficial.MontoDevolucion > 0 {
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(50, 6, tr("MONTO DEVUELTO:"), "", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		devoStr := fmt.Sprintf("Bs. %.2f (Boleta N° %s)", descargo.Oficial.MontoDevolucion, descargo.Oficial.NroBoletaDeposito)
		pdf.CellFormat(140, 6, tr(devoStr), "", 1, "L", false, 0, "")
	}

	// --- Signatures (Dynamic Position) ---
	currY := pdf.GetY()
	sigY := currY + 25

	// Ensure we don't start signatures too low on the page
	if sigY > 230 {
		pdf.AddPage()
		sigY = 40
	}

	pdf.SetY(sigY)
	pdf.SetFont("Arial", "B", 8)
	pdf.Line(35, sigY+10, 95, sigY+10)
	pdf.SetXY(35, sigY+12)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(60, 4, tr("SERVIDOR(A) PÚBLICO(A)"), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "", 7)
	pdf.CellFormat(60, 4, tr(solicitud.Usuario.GetNombreCompleto()), "", 1, "C", false, 0, "")
	pdf.SetX(35)
	pdf.SetFont("Arial", "I", 6)
	pdf.CellFormat(60, 3, tr("(FIRMA Y SELLO)"), "", 1, "C", false, 0, "")

	pdf.Line(110, sigY+10, 180, sigY+10)
	pdf.SetXY(110, sigY+12)
	pdf.SetFont("Arial", "B", 7)
	pdf.CellFormat(70, 4, tr("RESPONSABLE PRESENTACIÓN DEL DESCARGO"), "", 1, "C", false, 0, "")

	// Anexos (Imágenes)
	if descargo.Oficial != nil && len(descargo.Oficial.Anexos) > 0 {
		pdf.AddPage()
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(190, 10, tr("ANEXO FOTOGRÁFICO / ACTIVIDADES"), "B", 1, "C", false, 0, "")
		pdf.Ln(5)

		for _, anexo := range descargo.Oficial.Anexos {
			if _, err := os.Stat(anexo.Archivo); err == nil {
				// Evaluar si la imagen cabe en lo que queda de página, si no, nueva página
				// Asumimos un alto estimado o simplemente una por página para máxima claridad
				pdf.Image(anexo.Archivo, 15, pdf.GetY(), 180, 0, false, "", 0, "")
				pdf.Ln(5)
				// Dejar espacio o saltar página si ya está muy abajo
				if pdf.GetY() > 200 {
					pdf.AddPage()
				} else {
					pdf.Ln(10)
				}
			}
		}
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

	// 4. Unir usando pdftk
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

	// 4. Unir usando pdftk
	tmpFinal, err := os.CreateTemp("", "pv06_final_*.pdf")
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
