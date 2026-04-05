package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"
)

type DescargoOficialService struct {
	repo             *repositories.DescargoRepository
	solicitudService *SolicitudService
	auditService     *AuditService
}

func NewDescargoOficialService(
	repo *repositories.DescargoRepository,
	solicitudService *SolicitudService,
	auditService *AuditService,
) *DescargoOficialService {
	return &DescargoOficialService{
		repo:             repo,
		solicitudService: solicitudService,
		auditService:     auditService,
	}
}

func (s *DescargoOficialService) AutoCreateFromSolicitud(ctx context.Context, solicitud *models.Solicitud, userID string) (*models.Descargo, error) {
	existe, _ := s.repo.FindBySolicitudID(ctx, solicitud.ID)
	if existe != nil && existe.ID != "" {
		if err := s.SyncItineraryFromSolicitud(ctx, existe, solicitud); err != nil {
			return existe, err
		}
		return existe, nil
	}

	descargo := &models.Descargo{
		SolicitudID:       solicitud.ID,
		UsuarioID:         userID,
		Codigo:            solicitud.Codigo,
		FechaPresentacion: time.Now(),
		Estado:            models.EstadoDescargoBorrador,
	}
	descargo.CreatedBy = &userID

	descargo.Oficial = &models.DescargoOficial{
		DirigidoA: "SECRETARÍA GENERAL",
	}

	if err := s.repo.Create(ctx, descargo); err != nil {
		return nil, err
	}

	if err := s.SyncItineraryFromSolicitud(ctx, descargo, solicitud); err != nil {
		return descargo, err
	}

	s.auditService.Log(ctx, "AUTO_CREAR_DESCARGO_OFICIAL", "descargo", descargo.ID, "", string(models.EstadoDescargoBorrador), "Creado automáticamente desde solicitud (Oficial)", "")

	return descargo, nil
}

func (s *DescargoOficialService) SyncItineraryFromSolicitud(ctx context.Context, descargo *models.Descargo, solicitud *models.Solicitud) error {
	existingMap := make(map[string]bool)
	for _, det := range descargo.Tramos {
		if det.PasajeID != nil {
			existingMap[*det.PasajeID] = true
		}
	}

	modified := false
	process := func(item *models.SolicitudItem, tipoPrefix string) {
		if item == nil {
			return
		}
		for _, p := range item.Pasajes {
			st := p.GetEstadoCodigo()
			if st != "EMITIDO" && st != "USADO" {
				continue
			}

			tipo := tipoPrefix + "_ORIGINAL"

			tramosVuelo := p.GetTramosRuta()
			for range tramosVuelo {
				if !existingMap[p.ID] {
					tVuelo := p.FechaVuelo
					descargo.Tramos = append(descargo.Tramos, models.DescargoTramo{
						Tipo:            models.TipoDescargoTramo(tipo),
						RutaID:          p.RutaID,
						PasajeID:        &p.ID,
						SolicitudItemID: &item.ID,
						Fecha:           &tVuelo,
						Billete:         strings.ToUpper(strings.TrimSpace(p.NumeroBillete)),
					})
					existingMap[p.ID] = true
					modified = true
				}
			}
		}
	}

	process(solicitud.GetItemIda(), "IDA")
	process(solicitud.GetItemVuelta(), "VUELTA")

	if modified {
		return s.repo.Update(ctx, descargo)
	}
	return nil
}

func (s *DescargoOficialService) UpdateOficial(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, pasesAbordoPaths []string, anexoPaths []string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !descargo.IsEditable() {
		return fmt.Errorf("el descargo no se puede editar en su estado actual (%s)", descargo.Estado)
	}

	// 1. Basic Metadata
	descargo.Observaciones = req.Observaciones
	descargo.Estado = models.EstadoDescargoBorrador
	descargo.UpdatedBy = &userID

	// 2. Official Report (PV-06) Specific Data
	if descargo.Oficial == nil {
		descargo.Oficial = &models.DescargoOficial{DescargoID: id}
	}
	descargo.Oficial.NroMemorandum = req.NroMemorandum
	descargo.Oficial.ObjetivoViaje = req.ObjetivoViaje
	descargo.Oficial.InformeActividades = req.InformeActividades
	descargo.Oficial.ResultadosViaje = req.ResultadosViaje
	descargo.Oficial.ConclusionesRecomendaciones = req.ConclusionesRecomendaciones
	descargo.Oficial.MontoDevolucion = req.MontoDevolucion
	descargo.Oficial.NroBoletaDeposito = req.NroBoletaDeposito
	descargo.Oficial.DirigidoA = req.DirigidoA
	descargo.Oficial.TipoTransporte = req.TipoTransporte
	descargo.Oficial.PlacaVehiculo = req.PlacaVehiculo

	if err := s.repo.UpdateOficial(ctx, descargo.Oficial); err != nil {
		return err
	}

	// 3. Anexos & Terrestre Details
	if len(anexoPaths) > 0 {
		var anexos []models.AnexoDescargo
		for _, path := range anexoPaths {
			if path != "" {
				anexos = append(anexos, models.AnexoDescargo{DescargoOficialID: descargo.Oficial.ID, Archivo: path})
			}
		}
		s.repo.ClearAnexos(ctx, descargo.Oficial.ID)
		descargo.Oficial.Anexos = anexos
	}

	if len(req.TransporteTerrestreFecha) > 0 {
		var terrestres []models.TransporteTerrestreDescargo
		for i, fRaw := range req.TransporteTerrestreFecha {
			if fRaw != "" {
				terrestres = append(terrestres, models.TransporteTerrestreDescargo{
					DescargoOficialID: descargo.Oficial.ID,
					Fecha:             utils.ParseDate("2006-01-02", fRaw),
					NroFactura:        utils.GetIdx(req.TransporteTerrestreFactura, i),
					Importe:           utils.ParseFloat(utils.GetIdx(req.TransporteTerrestreImporte, i)),
					Tipo:              utils.GetIdxOrDefault(req.TransporteTerrestreTipo, i, "IDA"),
				})
			}
		}
		s.repo.ClearTransportesTerrestres(ctx, descargo.Oficial.ID)
		descargo.Oficial.TransportesTerrestres = terrestres
	}

	// 4. Data Cleansing & Mapping
	existingMap := make(map[string]models.DescargoTramo)
	for _, d := range descargo.Tramos {
		existingMap[d.ID] = d
	}

	// 5. Process Structured Itinerary Rows
	rows := req.ToTramoRows(pasesAbordoPaths)
	tramosProcesados := make([]models.DescargoTramo, 0, len(rows))

	for _, row := range rows {
		// Mandatory field check
		if row.RutaID == "" {
			continue
		}

		// Row ID preparation
		idRow := row.ID
		if strings.HasPrefix(idRow, "new_") {
			idRow = ""
		}

		// Field preparation
		fecha := utils.ParseDatePtr("2006-01-02", row.Fecha)
		rutaID := utils.NilIfEmpty(row.RutaID)
		pasajeID := utils.NilIfEmpty(row.PasajeID)
		solItemID := utils.NilIfEmpty(row.SolicitudItemID)

		// 6. Domain Rule: Data Protection for issued segments
		tipoRow := models.TipoDescargoTramo(row.Tipo)
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				// Fields from a pre-issued ticket segment are protected (read-only for business integrity)
				tipoRow = original.Tipo
				rutaID = original.RutaID
				fecha = original.Fecha
				pasajeID = original.PasajeID
				solItemID = original.SolicitudItemID
			}
		}

		// 7. Atomic Entity Assembly
		det := models.DescargoTramo{
			DescargoID:        id,
			Tipo:              tipoRow,
			RutaID:            rutaID,
			PasajeID:          pasajeID,
			SolicitudItemID:   solItemID,
			Fecha:             fecha,
			Billete:           row.Billete,
			NumeroPaseAbordo:  row.PaseNumero,
			ArchivoPaseAbordo: row.ArchivoPath,
			EsDevolucion:      row.EsDevolucion,
			EsModificacion:    row.EsModificacion,
			MontoDevolucion:   row.MontoDevolucion,
			Moneda:            row.Moneda,
		}
		det.ID = idRow
		tramosProcesados = append(tramosProcesados, det)
	}

	descargo.Tramos = tramosProcesados
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoOficialService) PrepareItinerarioOficial(descargo *models.Descargo) (map[string][]dtos.TramoView, map[string][]dtos.TramoView) {
	pasajesOriginales := make(map[string][]dtos.TramoView)
	pasajesReprogramados := make(map[string][]dtos.TramoView)

	itemsByType := make(map[string][]models.DescargoTramo)
	for _, item := range descargo.Tramos {
		itemsByType[string(item.Tipo)] = append(itemsByType[string(item.Tipo)], item)
	}

	tiposOrdenados := []string{"IDA_ORIGINAL", "IDA_REPRO", "VUELTA_ORIGINAL", "VUELTA_REPRO"}
	for _, tipo := range tiposOrdenados {
		items := itemsByType[tipo]
		for _, item := range items {
			dateStr := ""
			if item.Fecha != nil {
				dateStr = item.Fecha.Format("2006-01-02")
			}

			rutaStr := item.GetRutaDisplay()
			parts := strings.Split(rutaStr, " - ")
			rv := dtos.RutaView{Display: rutaStr}
			if len(parts) == 2 {
				rv.Origen = parts[0]
				rv.Destino = parts[1]
			} else {
				rv.Origen = rutaStr
			}

			cv := dtos.TramoView{
				ID:              item.ID,
				Tipo:            string(item.Tipo),
				Ruta:            rv,
				RutaID:          utils.DerefString(item.RutaID),
				Fecha:           dateStr,
				Billete:         item.Billete,
				Pase:            item.NumeroPaseAbordo,
				Archivo:         item.ArchivoPaseAbordo,
				EsDevolucion:    item.EsDevolucion,
				EsModificacion:  item.EsModificacion,
				MontoDevolucion: item.MontoDevolucion,
				Moneda:          item.Moneda,
				PasajeID:        utils.DerefString(item.PasajeID),
				SolicitudItemID: utils.DerefString(item.SolicitudItemID),
			}

			targetMap := pasajesOriginales
			if strings.HasSuffix(tipo, "REPRO") {
				targetMap = pasajesReprogramados
			}

			category := "IDA"
			if strings.HasPrefix(tipo, "VUELTA") {
				category = "VUELTA"
			}
			targetMap[category] = append(targetMap[category], cv)
		}
	}

	return pasajesOriginales, pasajesReprogramados
}
