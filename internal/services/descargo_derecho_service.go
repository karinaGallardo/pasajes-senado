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

type DescargoDerechoService struct {
	repo             *repositories.DescargoRepository
	solicitudService *SolicitudService
	auditService     *AuditService
}

func NewDescargoDerechoService(
	repo *repositories.DescargoRepository,
	solicitudService *SolicitudService,
	auditService *AuditService,
) *DescargoDerechoService {
	return &DescargoDerechoService{
		repo:             repo,
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

	// Indexar los tramos ORIGINALES ya existentes por PasajeID + Tipo con soporte multi-escala
	existingByKey := make(map[string][]models.DescargoTramo)
	for _, det := range descargo.Tramos {
		if det.PasajeID != nil && det.IsOriginal() {
			key := *det.PasajeID + "_" + string(det.Tipo)
			existingByKey[key] = append(existingByKey[key], det)
		}
	}

	for k, list := range existingByKey {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Seq < list[j].Seq
		})
		existingByKey[k] = list
	}

	tramosOriginalesNuevos := make([]models.DescargoTramo, 0, len(esperados))
	modified := false

	for _, esp := range esperados {
		key := *esp.PasajeID + "_" + string(esp.Tipo)
		list := existingByKey[key]

		if len(list) > 0 {
			existing := list[0]
			existingByKey[key] = list[1:]
			existing.RutaNombre = esp.RutaNombre
			tramosOriginalesNuevos = append(tramosOriginalesNuevos, existing)
		} else {
			// Nuevo → agregar directamente el modelo esperado
			tramosOriginalesNuevos = append(tramosOriginalesNuevos, esp)
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
	for _, det := range descargo.Tramos {
		if !det.IsOriginal() {
			tramosNoOriginales = append(tramosNoOriginales, det)
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
		tipoRow := models.TipoDescargoTramo(row.Tipo)
		rutaNombre := row.RutaNombre

		// 4. Domain Rule: Data Protection for issued segments
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
