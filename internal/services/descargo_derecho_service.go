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

type DescargoDerechoService struct {
	repo             *repositories.DescargoRepository
	rutaRepo         *repositories.RutaRepository
	solicitudService *SolicitudService
	auditService     *AuditService
}

func NewDescargoDerechoService(
	repo *repositories.DescargoRepository,
	rutaRepo *repositories.RutaRepository,
	solicitudService *SolicitudService,
	auditService *AuditService,
) *DescargoDerechoService {
	return &DescargoDerechoService{
		repo:             repo,
		rutaRepo:         rutaRepo,
		solicitudService: solicitudService,
		auditService:     auditService,
	}
}

func (s *DescargoDerechoService) GetShowData(ctx context.Context, id string) (*dtos.DescargoShowData, error) {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var itinerarioIda, itinerarioVuelta []models.DescargoTramo
	for _, item := range descargo.Tramos {
		if strings.HasPrefix(string(item.Tipo), "VUELTA") {
			itinerarioVuelta = append(itinerarioVuelta, item)
		} else {
			itinerarioIda = append(itinerarioIda, item)
		}
	}

	return &dtos.DescargoShowData{
		Descargo: descargo,
		Ida:      itinerarioIda,
		Vuelta:   itinerarioVuelta,
	}, nil
}

func (s *DescargoDerechoService) AutoCreateFromSolicitud(ctx context.Context, solicitud *models.Solicitud, userID string) (*models.Descargo, error) {
	// Verificar si ya existe
	existe, _ := s.repo.FindBySolicitudID(ctx, solicitud.ID)
	if existe != nil && existe.ID != "" {
		// Sincronizar por si hay nuevos pasajes emitidos después de la creación
		if err := s.SyncItineraryFromSolicitud(ctx, existe, solicitud); err != nil {
			return existe, err
		}
		return existe, nil
	}

	descargo := &models.Descargo{
		SolicitudID:       solicitud.ID,
		UsuarioID:         solicitud.UsuarioID, // El titular es el mismo que en la solicitud
		Codigo:            solicitud.Codigo,
		FechaPresentacion: time.Now(),
		Estado:            models.EstadoDescargoBorrador,
	}
	descargo.CreatedBy = &userID // El creador/operador es el usuario logueado

	if err := s.repo.Create(ctx, descargo); err != nil {
		return nil, err
	}

	// Poblar itinerario inicial
	if err := s.SyncItineraryFromSolicitud(ctx, descargo, solicitud); err != nil {
		return descargo, err
	}

	s.auditService.Log(ctx, "AUTO_CREAR_DESCARGO", "descargo", descargo.ID, "", string(models.EstadoDescargoBorrador), "Creado automáticamente desde solicitud", "")

	return descargo, nil
}

func (s *DescargoDerechoService) SyncItineraryFromSolicitud(ctx context.Context, descargo *models.Descargo, solicitud *models.Solicitud) error {
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

	// Indexar los tramos ORIGINALES ya existentes por PasajeID + Tipo + Ruta (Origen/Destino) con soporte de unicidad semántica
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

	for _, sobrantes := range existingByKey {
		if len(sobrantes) > 0 {
			modified = true
			break
		}
	}

	if !modified {
		return nil
	}

	tramosNoOriginales := make([]models.DescargoTramo, 0)
	for _, tramoGuardado := range descargo.Tramos {
		if !tramoGuardado.IsOriginal() {
			tramosNoOriginales = append(tramosNoOriginales, tramoGuardado)
		}
	}

	descargo.Tramos = append(tramosOriginalesNuevos, tramosNoOriginales...)
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoDerechoService) UpdateDerecho(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, pasesAbordoPaths []string) error {
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

	// 2. Data Cleansing & Mapping
	existingMap := make(map[string]models.DescargoTramo)
	for _, d := range descargo.Tramos {
		existingMap[d.ID] = d
	}

	// 3. Process Structured Itinerary Rows
	rows := req.ToTramoRows(pasesAbordoPaths)
	tramosProcesados := make([]models.DescargoTramo, 0, len(rows))

	// 3.1 Build City Map for Professional Itinerary Reconstruction
	allIATAs := make(map[string]bool)
	for _, row := range rows {
		if row.OrigenIATA != "" {
			allIATAs[row.OrigenIATA] = true
		}
		if row.DestinoIATA != "" {
			allIATAs[row.DestinoIATA] = true
		}
	}
	cityMap := make(map[string]string)
	if len(allIATAs) > 0 {
		var iataList []string
		for iata := range allIATAs {
			iataList = append(iataList, iata)
		}
		if destinos, err := s.rutaRepo.FindDestinosByIATAs(ctx, iataList); err == nil {
			for _, d := range destinos {
				cityMap[d.IATA] = d.GetNombreCorto()
			}
		}
	}

	// 3.2 Pre-fetch route displays
	routeMap := make(map[string]string)
	routeIDs := make([]string, 0)
	for _, r := range rows {
		if r.RutaID != "" {
			routeIDs = append(routeIDs, r.RutaID)
		}
	}
	if len(routeIDs) > 0 {
		for _, rid := range routeIDs {
			if r, err := s.rutaRepo.FindByID(ctx, rid); err == nil {
				routeMap[rid] = r.GetRutaDisplay()
			}
		}
	}

	for _, row := range rows {
		// Mandatory field check
		if row.RutaID == "" && (row.OrigenIATA == "" || row.DestinoIATA == "") {
			continue
		}

		// Row ID preparation
		idRow := row.ID
		if strings.HasPrefix(idRow, "new_") {
			idRow = ""
		}

		// Field preparation
		fecha, _ := utils.ParseDateTime(row.Fecha)
		rutaID := utils.NilIfEmpty(row.RutaID)
		pasajeID := utils.NilIfEmpty(row.PasajeID)
		solItemID := utils.NilIfEmpty(row.SolicitudItemID)
		origenIATA := utils.NilIfEmpty(row.OrigenIATA)
		destinoIATA := utils.NilIfEmpty(row.DestinoIATA)
		tipoRow := models.TipoDescargoTramo(row.Tipo)
		tramoNombre := row.TramoNombre

		// Professional Name Resolution ("Ciud (IATA) - Ciud (IATA)")
		if rutaID != nil {
			if fullName, ok := routeMap[*rutaID]; ok && fullName != "" {
				tramoNombre = fullName
			}
		}

		// Fallback for manual/reprogrammed segments: Reconstruct from cityMap
		if tramoNombre == "" || !strings.Contains(tramoNombre, "(") {
			if origenIATA != nil && destinoIATA != nil {
				oLabel := *origenIATA
				if city, ok := cityMap[*origenIATA]; ok {
					oLabel = city
				}
				dLabel := *destinoIATA
				if city, ok := cityMap[*destinoIATA]; ok {
					dLabel = city
				}
				tramoNombre = oLabel + " - " + dLabel
			}
		}

		// 4. Domain Rule: Data Protection for issued segments
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				// Fields from a pre-issued ticket segment are protected
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
					NumeroVuelo:       original.NumeroVuelo,
					NumeroPaseAbordo:  row.PaseNumero,
					ArchivoPaseAbordo: row.ArchivoPath,
					EsDevolucion:      row.EsDevolucion,
					EsModificacion:    row.EsModificacion,
					TramoNombre:       original.TramoNombre,
					Seq:               row.Seq,
				}
				tramosProcesados = append(tramosProcesados, det)
				continue
			}
		}

		// 5. Atomic Entity Assembly
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
			TramoNombre:       tramoNombre,
			Seq:               row.Seq,
		}
		tramosProcesados = append(tramosProcesados, det)
	}

	descargo.Tramos = tramosProcesados
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoDerechoService) GetEditData(ctx context.Context, id string) (*dtos.DescargoEditData, error) {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if !descargo.IsEditable() {
		return nil, fmt.Errorf("el descargo no se puede editar en su estado actual (%s)", descargo.Estado)
	}

	// Sincronización proactiva: Si se emitieron nuevos pasajes después de la creación inicial
	if descargo.Solicitud != nil {
		if err := s.SyncItineraryFromSolicitud(ctx, descargo, descargo.Solicitud); err == nil {
			// Recargar para relacionales GORM si hubo cambios
			descargo, _ = s.repo.FindByID(ctx, id)
		}
	}

	// 2. Agrupar por categoría (IDA/VUELTA) pero manteniendo la estructura para el formulario
	pasajesIda := make([]models.DescargoTramo, 0)
	pasajesVuelta := make([]models.DescargoTramo, 0)

	for _, item := range descargo.Tramos {
		if strings.HasPrefix(string(item.Tipo), "VUELTA") {
			pasajesVuelta = append(pasajesVuelta, item)
		} else {
			pasajesIda = append(pasajesIda, item)
		}
	}

	return &dtos.DescargoEditData{
		Descargo:  descargo,
		Solicitud: descargo.Solicitud,
		Ida:       pasajesIda,
		Vuelta:    pasajesVuelta,
	}, nil
}
