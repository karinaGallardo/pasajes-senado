package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

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
		"N°", "FECHA EMISIÓN", "FECHA VUELO", "HORA VUELO", "N° VUELO", "IDA/VUELTA", "CÓDIGO SOL.", "CONCEPTO", "TIPO DE SOLICITUD", "ÁMBITO", "BENEFICIARIO", "TIPO PASAJERO", "UNIDAD ORGANIZACIONAL", "CARGO", "ORIGEN", "DESTINO", "RUTA / TRAMOS", "AEROLÍNEA", "AGENCIA", "NRO. BILLETE",
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
				descargoEstado = strings.ToUpper(models.EstadoDescargoStatusInfo(sol.Descargo.Estado).Nombre)
			}
		}

		beneficiario := "-"
		unidadOrg := "-"
		cargo := "-"
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			u := p.SolicitudItem.Solicitud.Usuario
			beneficiario = u.GetNombreCompleto()
			if u.Oficina != nil {
				unidadOrg = u.Oficina.Detalle
			}
			if u.Cargo != nil {
				cargo = u.Cargo.Descripcion
			}
		}

		montoCargos := p.GetMontoCargos()
		costoTotalPasaje := p.Costo + montoCargos

		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		if p.FechaEmision != nil {
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), p.FechaEmision.Format("02/01/2006"))
		}

		fechaVuelo := "-"
		if !p.FechaVuelo.IsZero() {
			fechaVuelo = p.FechaVuelo.Format("02/01/2006")
		}
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fechaVuelo)

		horaVuelo := "-"
		if !p.FechaVuelo.IsZero() {
			horaVuelo = p.FechaVuelo.Format("15:04")
		}
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), horaVuelo)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), p.NumeroVuelo)

		tipoVuelo := "-"
		if p.SolicitudItem != nil {
			tipoVuelo = string(p.SolicitudItem.Tipo)
		}
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), tipoVuelo)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), codigoSolicitud)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), concepto)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), tipoSolicitud)

		ambito := "-"
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			ambito = p.SolicitudItem.Solicitud.AmbitoViajeCodigo
		}
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), ambito)

		f.SetCellValue(sheet, fmt.Sprintf("K%d", row), beneficiario)

		tipoBeneficiario := "FUNCIONARIO"
		if p.SolicitudItem != nil && p.SolicitudItem.Solicitud != nil {
			if p.SolicitudItem.Solicitud.Usuario.IsSenador() {
				tipoBeneficiario = "SENADOR"
			}
		}
		f.SetCellValue(sheet, fmt.Sprintf("L%d", row), tipoBeneficiario)

		f.SetCellValue(sheet, fmt.Sprintf("M%d", row), unidadOrg)
		f.SetCellValue(sheet, fmt.Sprintf("N%d", row), cargo)

		origen := "-"
		destino := "-"
		if p.RutaPasaje != nil {
			if p.RutaPasaje.Origen.IATA != "" {
				origen = p.RutaPasaje.Origen.GetNombreCorto()
			} else {
				origen = p.RutaPasaje.OrigenIATA
			}

			if p.RutaPasaje.Destino.IATA != "" {
				destino = p.RutaPasaje.Destino.GetNombreCorto()
			} else {
				destino = p.RutaPasaje.DestinoIATA
			}
		}
		f.SetCellValue(sheet, fmt.Sprintf("O%d", row), origen)
		f.SetCellValue(sheet, fmt.Sprintf("P%d", row), destino)

		f.SetCellValue(sheet, fmt.Sprintf("Q%d", row), p.GetRutaDisplay())

		if p.Aerolinea != nil {
			f.SetCellValue(sheet, fmt.Sprintf("R%d", row), p.Aerolinea.Sigla)
		}
		if p.Agencia != nil {
			f.SetCellValue(sheet, fmt.Sprintf("S%d", row), p.Agencia.Nombre)
		}

		f.SetCellValue(sheet, fmt.Sprintf("T%d", row), p.NumeroBillete)
		f.SetCellValue(sheet, fmt.Sprintf("U%d", row), p.Costo)
		f.SetCellValue(sheet, fmt.Sprintf("V%d", row), p.MontoReembolso)
		f.SetCellValue(sheet, fmt.Sprintf("W%d", row), p.CostoUtilizado)
		f.SetCellValue(sheet, fmt.Sprintf("X%d", row), montoCargos)
		f.SetCellValue(sheet, fmt.Sprintf("Y%d", row), costoTotalPasaje)
		f.SetCellValue(sheet, fmt.Sprintf("Z%d", row), p.GetEstado())
		f.SetCellValue(sheet, fmt.Sprintf("AA%d", row), solicitudEstado)
		f.SetCellValue(sheet, fmt.Sprintf("AB%d", row), descargoEstado)

		totalCostoOrigen += p.Costo
		totalCargos += montoCargos
		totalGeneral += costoTotalPasaje

		// Aplicar estilo de borde a la fila
		lastCell, _ := excelize.CoordinatesToCellName(len(headers), row)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), lastCell, rowStyle)
	}

	// Fila de Totales
	totalRow := len(pasajes) + 2
	f.SetCellValue(sheet, fmt.Sprintf("T%d", totalRow), "TOTALES:")
	f.SetCellValue(sheet, fmt.Sprintf("U%d", totalRow), totalCostoOrigen)
	f.SetCellValue(sheet, fmt.Sprintf("X%d", totalRow), totalCargos)
	f.SetCellValue(sheet, fmt.Sprintf("Y%d", totalRow), totalGeneral)

	totalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"F3F4F6"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right"},
	})
	f.SetCellStyle(sheet, fmt.Sprintf("T%d", totalRow), fmt.Sprintf("Y%d", totalRow), totalStyle)

	// Autoajustar anchos (aproximado)
	widths := []float64{
		5,  // A: N°
		15, // B: FECHA EMISIÓN
		15, // C: FECHA VUELO
		12, // D: HORA VUELO
		12, // E: N° VUELO
		15, // F: IDA/VUELTA
		15, // G: CÓDIGO SOL.
		12, // H: CONCEPTO
		20, // I: TIPO DE SOLICITUD
		15, // J: ÁMBITO
		35, // K: BENEFICIARIO
		18, // L: TIPO PASAJERO
		30, // M: UNIDAD ORGANIZACIONAL
		30, // N: CARGO
		25, // O: ORIGEN
		25, // P: DESTINO
		45, // Q: RUTA / TRAMOS
		12, // R: AEROLÍNEA
		25, // S: AGENCIA
		18, // T: NRO. BILLETE
		18, // U: COSTO ORIGEN (BS)
		15, // V: DEV DIF TARIFA
		15, // W: COSTO CONSUMO
		18, // X: CARGOS ASOCIADOS
		18, // Y: COSTO TOTAL
		15, // Z: ESTADO PASAJE
		18, // AA: ESTADO SOLICITUD
		18, // AB: ESTADO DESCARGO
	}
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
