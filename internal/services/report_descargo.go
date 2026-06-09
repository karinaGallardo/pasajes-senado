package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
)

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
	pdf.CellFormat(190, 6, tr(" TRAMOS NO UTILIZADOS (OPEN TICKETS UTILIZABLES)"), "B", 1, "L", false, 0, "")
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
	s.drawReportHeader(pdf, tr, "FORM-PV-05", "REPORTE DE BILLETES PARA UTILIZACIÓN", "CÁMARA DE SENADORES", "GESTIÓN: "+gestion, descargo.Codigo)

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
	pdf.CellFormat(0, 8, tr("1. DETALLE DE UTILIZACIÓN DE BILLETES (MOVIMIENTOS)"), "B", 1, "L", false, 0, "")
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
				pdf.CellFormat(40, 7, tr("UTILIZADO"), "1", 1, "C", true, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Arial", "", 8)
			}
		}
		pdf.Ln(3)
	}

	if !hasAnyOT {
		pdf.SetFont("Arial", "", 8)
		pdf.CellFormat(190, 8, tr("No se han registrado movimientos de utilización en este descargo."), "1", 1, "C", false, 0, "")
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
	pdf.MultiCell(190, 5, tr("Se certifica que los tramos detallados corresponden a pasajes previamente devueltos y no utilizados, que fueron pagados en su emisión original y posteriormente utilizados por el beneficiario, conforme a normativa vigente."), "", "L", false)

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
