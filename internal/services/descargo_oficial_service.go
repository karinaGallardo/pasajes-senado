package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"sort"
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
	esperados := make([]models.DescargoTramo, 0)
	seqCounter := 1

	buildEsperados := func(item *models.SolicitudItem, tipoPrefix string) {
		if item == nil {
			return
		}
		for _, p := range item.Pasajes {
			if !p.IsDischargeable() {
				continue
			}
			tipo := models.TipoDescargoTramo(tipoPrefix + "_ORIGINAL")
			legs := p.GetTramosLegs()
			for _, leg := range legs {
				tVuelo := p.FechaVuelo
				pID := p.ID
				sID := item.ID
				orig := leg.OrigenIATA
				dest := leg.DestinoIATA

				esperados = append(esperados, models.DescargoTramo{
					DescargoID:      descargo.ID,
					PasajeID:        &pID,
					RutaID:          p.RutaID,
					Tipo:            tipo,
					SolicitudItemID: &sID,
					Fecha:           &tVuelo,
					Billete:         strings.ToUpper(strings.TrimSpace(p.NumeroBillete)),
					NumeroVuelo:     p.NumeroVuelo,
					OrigenIATA:      &orig,
					DestinoIATA:     &dest,
					RutaNombre:      orig + " - " + dest,
					Seq:             seqCounter,
				})
				seqCounter++
			}
		}
	}

	for _, item := range solicitud.GetAllItemsIda() {
		buildEsperados(item, "IDA")
	}
	for _, item := range solicitud.GetAllItemsVuelta() {
		buildEsperados(item, "VUELTA")
	}

	// Indexar los tramos ORIGINALES ya existentes por PasajeID + Tipo
	// En lugar de mapa plano, usamos slice para soportar pasajes con múltiples escalas
	existingByKey := make(map[string][]models.DescargoTramo)
	for _, det := range descargo.Tramos {
		if det.PasajeID != nil && det.IsOriginal() {
			key := *det.PasajeID + "_" + string(det.Tipo)
			existingByKey[key] = append(existingByKey[key], det)
		}
	}

	// Ordenar las listas por ID o Seq para que el consumo sea determinista
	for k, list := range existingByKey {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Seq < list[j].Seq
		})
		existingByKey[k] = list
	}

	// Construir el nuevo slice de tramos originales fusionando existentes con esperados
	tramosOriginalesNuevos := make([]models.DescargoTramo, 0, len(esperados))
	modified := false

	for _, esp := range esperados {
		key := *esp.PasajeID + "_" + string(esp.Tipo)
		list := existingByKey[key]

		if len(list) > 0 {
			// Consumir el primero de la cola (FIFO)
			existing := list[0]
			existingByKey[key] = list[1:]

			// Restaurar el campo volátil (no persistido en DB) para el ViewModel/Template
			existing.RutaNombre = esp.RutaNombre

			tramosOriginalesNuevos = append(tramosOriginalesNuevos, existing)
		} else {
			// Nuevo → agregar directamente el modelo esperado
			tramosOriginalesNuevos = append(tramosOriginalesNuevos, esp)
			modified = true
		}
	}

	// Los tramos que quedaron huérfanos en las colas no corresponden a pasajes emitidos → se perderán
	for _, sobrantes := range existingByKey {
		if len(sobrantes) > 0 {
			modified = true
			break
		}
	}

	if !modified {
		return nil
	}

	// Reconstruir el slice completo: tramos originales sincronizados + reprogramados + devoluciones (sin cambios)
	tramosNoOriginales := make([]models.DescargoTramo, 0)
	for _, det := range descargo.Tramos {
		if !det.IsOriginal() {
			tramosNoOriginales = append(tramosNoOriginales, det)
		}
	}

	descargo.Tramos = append(tramosOriginalesNuevos, tramosNoOriginales...)
	return s.repo.Update(ctx, descargo)
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
	if descargo.Estado == models.EstadoDescargoRechazado {
		descargo.Estado = models.EstadoDescargoEnRevision
	} else {
		descargo.Estado = models.EstadoDescargoBorrador
	}
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
		origenIATA := utils.NilIfEmpty(row.OrigenIATA)
		destinoIATA := utils.NilIfEmpty(row.DestinoIATA)
		rutaNombre := row.RutaNombre

		// 6. Domain Rule: Data Protection for issued segments
		tipoRow := models.TipoDescargoTramo(row.Tipo)
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				// Fields from a pre-issued ticket segment are protected (read-only for business integrity)
				tipoRow = original.Tipo
				rutaID = original.RutaID
				rutaNombre = original.RutaNombre
				origenIATA = original.OrigenIATA
				destinoIATA = original.DestinoIATA
				vuelo := original.NumeroVuelo
				fecha = original.Fecha
				pasajeID = original.PasajeID
				solItemID = original.SolicitudItemID

				det := models.DescargoTramo{
					BaseModel:         models.BaseModel{ID: idRow},
					DescargoID:        id,
					Tipo:              tipoRow,
					RutaID:            rutaID,
					PasajeID:          pasajeID,
					SolicitudItemID:   solItemID,
					OrigenIATA:        origenIATA,
					DestinoIATA:       destinoIATA,
					Fecha:             fecha,
					Billete:           row.Billete,
					NumeroVuelo:       vuelo,
					NumeroPaseAbordo:  row.PaseNumero,
					ArchivoPaseAbordo: row.ArchivoPath,
					EsDevolucion:      row.EsDevolucion,
					EsModificacion:    row.EsModificacion,
					MontoDevolucion:   row.MontoDevolucion,
					Moneda:            row.Moneda,
					RutaNombre:        rutaNombre,
					Seq:               row.Seq,
				}
				tramosProcesados = append(tramosProcesados, det)
				continue
			}
		}

		// 7. Atomic Entity Assembly
		det := models.DescargoTramo{
			BaseModel:         models.BaseModel{ID: idRow},
			DescargoID:        id,
			Tipo:              tipoRow,
			RutaID:            rutaID,
			PasajeID:          pasajeID,
			SolicitudItemID:   solItemID,
			OrigenIATA:        origenIATA,
			DestinoIATA:       destinoIATA,
			Fecha:             fecha,
			Billete:           row.Billete,
			NumeroVuelo:       row.Vuelo,
			NumeroPaseAbordo:  row.PaseNumero,
			ArchivoPaseAbordo: row.ArchivoPath,
			EsDevolucion:      row.EsDevolucion,
			EsModificacion:    row.EsModificacion,
			MontoDevolucion:   row.MontoDevolucion,
			Moneda:            row.Moneda,
			RutaNombre:        rutaNombre,
			Seq:               row.Seq,
		}
		det.ID = idRow
		tramosProcesados = append(tramosProcesados, det)
	}

	descargo.Tramos = tramosProcesados
	return s.repo.Update(ctx, descargo)
}
