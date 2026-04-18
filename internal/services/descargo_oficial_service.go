package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"
	"time"
)

type DescargoOficialService struct {
	repo             *repositories.DescargoRepository
	rutaRepo         *repositories.RutaRepository
	descargoService  *DescargoService
	solicitudService *SolicitudService
	auditService     *AuditService
	pasajeRepo       *repositories.PasajeRepository
}

func NewDescargoOficialService(
	repo *repositories.DescargoRepository,
	rutaRepo *repositories.RutaRepository,
	descargoService *DescargoService,
	solicitudService *SolicitudService,
	auditService *AuditService,
	pasajeRepo *repositories.PasajeRepository,
) *DescargoOficialService {
	return &DescargoOficialService{
		repo:             repo,
		rutaRepo:         rutaRepo,
		descargoService:  descargoService,
		solicitudService: solicitudService,
		auditService:     auditService,
		pasajeRepo:       pasajeRepo,
	}
}

func (s *DescargoOficialService) GetShowData(ctx context.Context, id string) (*models.Descargo, error) {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Sincronización proactiva: Si se emitieron nuevos pasajes
	if descargo.Solicitud != nil && descargo.IsEditable() {
		if err := s.SyncItineraryFromSolicitud(ctx, descargo, descargo.Solicitud); err == nil {
			// Recargar para tener tramos pre-cargados
			descargo, _ = s.repo.FindByID(ctx, id)
		}
	}

	return descargo, nil
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
		DirigidoA:     "",
		NroMemorandum: solicitud.Autorizacion,
		ObjetivoViaje: solicitud.Motivo,
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
	tramosPasajesEmitidos := make([]models.DescargoTramo, 0)
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

				tramosPasajesEmitidos = append(tramosPasajesEmitidos, models.DescargoTramo{
					DescargoID:      descargo.ID,
					PasajeID:        &pID,
					RutaID:          p.RutaID,
					Tipo:            tipo,
					SolicitudItemID: &sID,
					Fecha:           &tVuelo,
					Billete:         strings.ToUpper(strings.TrimSpace(p.NumeroBillete)),
					NumeroVuelo:     "",
					OrigenIATA:      &orig,
					DestinoIATA:     &dest,
					TramoNombre:     leg.GetLabel(),
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

	// Indexar los tramos ORIGINALES ya existentes por PasajeID + Tipo + Ruta (Origen/Destino)
	// Esta llave atómica asegura la unicidad semántica de cada pierna del viaje (tramoGuardado)
	existingByKey := make(map[string][]models.DescargoTramo)
	for _, tramoGuardado := range descargo.Tramos {
		if tramoGuardado.PasajeID != nil && tramoGuardado.IsOriginal() {
			key := fmt.Sprintf("%s_%s_%s_%s", *tramoGuardado.PasajeID, string(tramoGuardado.Tipo), tramoGuardado.GetOrigenIATA(), tramoGuardado.GetDestinoIATA())
			existingByKey[key] = append(existingByKey[key], tramoGuardado)
		}
	}

	// Construir el nuevo slice de tramos originales fusionando existentes con proyectados
	tramosOriginalesNuevos := make([]models.DescargoTramo, 0, len(tramosPasajesEmitidos))
	modified := false

	for _, tramoEmitido := range tramosPasajesEmitidos {
		key := fmt.Sprintf("%s_%s_%s_%s", *tramoEmitido.PasajeID, string(tramoEmitido.Tipo), tramoEmitido.GetOrigenIATA(), tramoEmitido.GetDestinoIATA())
		list := existingByKey[key]

		if len(list) > 0 {
			// Consumir el primero de la cola (FIFO)
			existing := list[0]
			existingByKey[key] = list[1:]

			// Restaurar el campo volátil (no persistido en DB) para el ViewModel/Template
			existing.TramoNombre = tramoEmitido.TramoNombre

			tramosOriginalesNuevos = append(tramosOriginalesNuevos, existing)
		} else {
			// Nuevo → agregar directamente el modelo emitido
			tramosOriginalesNuevos = append(tramosOriginalesNuevos, tramoEmitido)
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
	for _, tramoGuardado := range descargo.Tramos {
		if !tramoGuardado.IsOriginal() {
			tramosNoOriginales = append(tramosNoOriginales, tramoGuardado)
		}
	}

	descargo.Tramos = append(tramosOriginalesNuevos, tramosNoOriginales...)
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoOficialService) UpdateOficial(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, pasesAbordoPaths []string, terrestrePaths []string, anexoPaths []string, boletasPaths []string, memoPath string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !descargo.IsEditable() {
		return fmt.Errorf("el descargo no se puede editar en su estado actual (%s)", descargo.Estado)
	}

	// 1. Basic Metadata
	descargo.Observaciones = req.Observaciones
	if req.FechaPresentacion != "" {
		if fp, err := utils.ParseDateTime(req.FechaPresentacion); err == nil && fp != nil {
			descargo.FechaPresentacion = *fp
		}
	}
	// No cambiamos el estado automáticamente. El usuario debe "Enviar a Revisión" explícitamente.
	if descargo.Estado == "" {
		descargo.Estado = models.EstadoDescargoBorrador
	}
	descargo.UpdatedBy = &userID

	// 1.1 Comprobantes de Pago (Ahora per pasaje)
	totalDevolucion := 0.0
	for i, pasajeID := range req.LiquidacionPasajeID {
		monto := 0.0
		if i < len(req.LiquidacionMontoDevolucion) {
			monto, _ = strconv.ParseFloat(req.LiquidacionMontoDevolucion[i], 64)
		}
		totalDevolucion += monto

		nroBoleta := ""
		if i < len(req.LiquidacionNroBoleta) {
			nroBoleta = req.LiquidacionNroBoleta[i]
		}

		boletaPath := ""
		if i < len(boletasPaths) {
			boletaPath = boletasPaths[i]
		}

		// Buscar el pasaje original y actualizar campos financieros
		if p, err := s.pasajeRepo.FindByID(ctx, pasajeID); err == nil {
			p.MontoReembolso = monto
			p.CostoUtilizado = p.Costo - monto
			p.NroBoletaDeposito = nroBoleta
			if boletaPath != "" {
				p.ArchivoComprobante = boletaPath
				now := time.Now()
				p.FechaDeposito = &now
			}
			_ = s.pasajeRepo.Update(ctx, p)
		}
	}

	// 2. Official Report (PV-06) Specific Data
	if descargo.Oficial == nil {
		descargo.Oficial = &models.DescargoOficial{DescargoID: id}
	}
	descargo.Oficial.NroMemorandum = req.NroMemorandum
	descargo.Oficial.ObjetivoViaje = req.ObjetivoViaje
	descargo.Oficial.InformeActividades = req.InformeActividades
	descargo.Oficial.ResultadosViaje = req.ResultadosViaje
	descargo.Oficial.ConclusionesRecomendaciones = req.ConclusionesRecomendaciones
	descargo.Oficial.NroBoletaDeposito = req.NroBoletaDeposito
	descargo.Oficial.DirigidoA = req.DirigidoA
	descargo.Oficial.LugarViaje = req.LugarViaje
	descargo.Oficial.TipoTransporte = req.TipoTransporte
	descargo.Oficial.PlacaVehiculo = req.PlacaVehiculo
	descargo.Oficial.ArchivoMemorandum = memoPath

	if err := s.repo.UpdateOficial(ctx, descargo.Oficial); err != nil {
		return err
	}

	// Forzar ID si fue una creación nueva (el repo Save suele inyectarlo, pero blindamos)
	oficialID := descargo.Oficial.ID
	if oficialID == "" {
		// Fallback por si la relación aún no está sincronizada en memoria
		updatedOficial, _ := s.repo.FindOficialByDescargoID(ctx, id)
		if updatedOficial != nil {
			oficialID = updatedOficial.ID
			descargo.Oficial = updatedOficial
		}
	}

	// 3. Anexos
	s.repo.ClearAnexos(ctx, oficialID)
	if len(anexoPaths) > 0 {
		var anexos []models.AnexoDescargo
		for _, path := range anexoPaths {
			if path != "" {
				anexos = append(anexos, models.AnexoDescargo{DescargoOficialID: oficialID, Archivo: path})
			}
		}
		if len(anexos) > 0 {
			if err := s.repo.SaveAnexos(ctx, anexos); err != nil {
				return err
			}
		}
		descargo.Oficial.Anexos = anexos
	}

	// 4. Transportes Terrestres
	s.repo.ClearTransportesTerrestres(ctx, oficialID)
	if len(req.TransporteTerrestreFecha) > 0 {
		var terrestres []models.TransporteTerrestreDescargo
		for i, fRaw := range req.TransporteTerrestreFecha {
			if fRaw != "" {
				fechaTerrestre, _ := utils.ParseDateTime(fRaw)
				if fechaTerrestre != nil {
					terrestres = append(terrestres, models.TransporteTerrestreDescargo{
						DescargoOficialID: oficialID,
						Fecha:             *fechaTerrestre,
						NroFactura:        utils.GetIdx(req.TransporteTerrestreFactura, i),
						Importe:           utils.ParseFloat(utils.GetIdx(req.TransporteTerrestreImporte, i)),
						Tipo:              utils.GetIdxOrDefault(req.TransporteTerrestreTipo, i, "IDA"),
						Archivo:           utils.GetIdx(terrestrePaths, i),
					})
				}
			}
		}
		if len(terrestres) > 0 {
			if err := s.repo.SaveTransportesTerrestres(ctx, terrestres); err != nil {
				return err
			}
		}
		descargo.Oficial.TransportesTerrestres = terrestres
	}

	// 4. Liquidación de Pasajes (Actualización Directa en Modelo Pasaje)
	if len(req.LiquidacionPasajeID) > 0 {
		totalDevo := 0.0
		for i, pID := range req.LiquidacionPasajeID {
			if pID != "" {
				montoDevo := utils.ParseFloat(utils.GetIdx(req.LiquidacionMontoDevolucion, i))
				totalDevo += montoDevo

				// Auto-calcular costo utilizado recuperando el original
				var p models.Pasaje
				if err := s.repo.GetDB().Select("id", "costo").First(&p, "id = ?", pID).Error; err == nil {
					costoUtilizado := p.Costo - montoDevo
					if err := s.repo.GetDB().Model(&models.Pasaje{}).Where("id = ?", pID).Updates(map[string]interface{}{
						"costo_utilizado": costoUtilizado,
						"monto_reembolso": montoDevo,
					}).Error; err != nil {
						fmt.Printf("[ERROR] Fallo al actualizar liquidación pasaje %s: %v\n", pID, err)
					}
				}
			}
		}
	}

	// 5. Data Cleansing & Mapping
	existingMap := make(map[string]models.DescargoTramo)
	for _, d := range descargo.Tramos {
		existingMap[d.ID] = d
	}

	// 5. Build Itinerary with Professional Names
	rows := req.ToTramoRows(pasesAbordoPaths)
	uniqueIATAs := make(map[string]bool)
	for _, r := range rows {
		if r.OrigenIATA != "" {
			uniqueIATAs[r.OrigenIATA] = true
		}
		if r.DestinoIATA != "" {
			uniqueIATAs[r.DestinoIATA] = true
		}
	}

	cityMap := make(map[string]string)
	if len(uniqueIATAs) > 0 {
		var list []string
		for k := range uniqueIATAs {
			list = append(list, k)
		}
		if destinos, err := s.rutaRepo.FindDestinosByIATAs(ctx, list); err == nil {
			for _, d := range destinos {
				cityMap[d.IATA] = d.GetNombreCorto()
			}
		}
	}

	tramosProcesados := make([]models.DescargoTramo, 0, len(rows))
	for _, row := range rows {
		if row.OrigenIATA == "" || row.DestinoIATA == "" {
			continue
		}

		// Resolution of Professional Name: "City (IATA) - City (IATA)"
		oLabel := row.OrigenIATA
		if name, ok := cityMap[row.OrigenIATA]; ok {
			oLabel = name
		}
		dLabel := row.DestinoIATA
		if name, ok := cityMap[row.DestinoIATA]; ok {
			dLabel = name
		}
		tramoNombre := oLabel + " - " + dLabel

		// Row ID & Data Prep
		idRow := row.ID
		if strings.HasPrefix(idRow, "new_") {
			idRow = ""
		}
		fecha, _ := utils.ParseDateTime(row.Fecha)
		rutaID := utils.NilIfEmpty(row.RutaID)
		pasajeID := utils.NilIfEmpty(row.PasajeID)
		solItemID := utils.NilIfEmpty(row.SolicitudItemID)

		// Domain Rule: Protect original/issued segments
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				det := models.DescargoTramo{
					BaseModel:         models.BaseModel{ID: idRow},
					DescargoID:        id,
					Tipo:              original.Tipo,
					RutaID:            original.RutaID,
					PasajeID:          original.PasajeID,
					SolicitudItemID:   original.SolicitudItemID,
					OrigenIATA:        original.OrigenIATA,
					DestinoIATA:       original.DestinoIATA,
					Fecha:             original.Fecha,
					Billete:           row.Billete,
					NumeroVuelo:       row.Vuelo,
					NumeroPaseAbordo:  row.PaseNumero,
					ArchivoPaseAbordo: row.ArchivoPath,
					EsOpenTicket:      row.EsOpenTicket,
					EsModificacion:    row.EsModificacion,
					TramoNombre:       original.TramoNombre,
					Seq:               row.Seq,
				}
				tramosProcesados = append(tramosProcesados, det)
				continue
			}
		}

		// New/Manual/Reprogrammed Entity
		det := models.DescargoTramo{
			BaseModel:         models.BaseModel{ID: idRow},
			DescargoID:        id,
			Tipo:              models.TipoDescargoTramo(row.Tipo),
			RutaID:            rutaID,
			PasajeID:          pasajeID,
			SolicitudItemID:   solItemID,
			OrigenIATA:        &row.OrigenIATA,
			DestinoIATA:       &row.DestinoIATA,
			Fecha:             fecha,
			Billete:           row.Billete,
			NumeroVuelo:       row.Vuelo,
			NumeroPaseAbordo:  row.PaseNumero,
			ArchivoPaseAbordo: row.ArchivoPath,
			EsOpenTicket:      row.EsOpenTicket,
			EsModificacion:    row.EsModificacion,
			TramoNombre:       tramoNombre,
			Seq:               row.Seq,
		}
		tramosProcesados = append(tramosProcesados, det)
	}

	descargo.Tramos = tramosProcesados
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	return nil
}
