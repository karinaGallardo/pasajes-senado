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

	// 1. Obtener tramos y sincronizar virtualmente con la solicitud
	tramos := descargo.Tramos
	if descargo.Solicitud != nil {
		tramos = s.syncVirtualItinerary(tramos, descargo.Solicitud)
	}

	// 2. Agrupar por Billete para la UI
	ticketsMap := make(map[string]*dtos.BilleteItinerario)
	var ticketsOrder []string

	for _, d := range tramos {
		key := d.Billete
		if key == "" {
			key = "SIN_BILLETE"
		}

		if _, ok := ticketsMap[key]; !ok {
			ticketsMap[key] = &dtos.BilleteItinerario{Billete: d.Billete}
			ticketsOrder = append(ticketsOrder, key)
		}

		tg := ticketsMap[key]
		tg.Tramos = append(tg.Tramos, d)
		if d.EsDevolucion {
			// ticketsMap[key].EsDevolucion = true // No longer needed at grouping level
		}
		if d.EsModificacion {
			// ticketsMap[key].EsModificacion = true // No longer needed at grouping level
		}
	}

	// 3. Clasificar en IDA y VUELTA
	var billetesIda, billetesVuelta []dtos.BilleteItinerario
	for _, key := range ticketsOrder {
		tg := ticketsMap[key]

		// Ordenar tramos por el campo Orden
		// sort.Slice is safe here.
		s.sortTramos(tg.Tramos)

		// Clasificar por el tipo del primer tramo
		if len(tg.Tramos) > 0 && strings.HasPrefix(string(tg.Tramos[0].Tipo), "VUELTA") {
			billetesVuelta = append(billetesVuelta, *tg)
		} else {
			billetesIda = append(billetesIda, *tg)
		}
	}

	return &dtos.DescargoShowData{
		Descargo: descargo,
		Ida:      billetesIda,
		Vuelta:   billetesVuelta,
	}, nil
}

func (s *DescargoDerechoService) syncVirtualItinerary(
	actuales []models.DescargoTramo,
	solicitud *models.Solicitud,
) []models.DescargoTramo {
	existingKeys := make(map[string]bool)
	for _, it := range actuales {
		key := fmt.Sprintf("%s_%d_%s", it.Tipo, it.Orden, strings.ToUpper(it.Billete))
		existingKeys[key] = true
	}

	for _, sItem := range solicitud.Items {
		tipoBase := string(sItem.Tipo)
		for _, p := range sItem.Pasajes {
			if !p.IsDischargeable() {
				continue
			}

			tipoTarget := tipoBase + "_ORIGINAL"

			tramosVuelo := p.GetTramosRuta()
			for i := range tramosVuelo {
				key := fmt.Sprintf("%s_%d_%s", tipoTarget, i, strings.ToUpper(p.NumeroBillete))
				if !existingKeys[key] {
					tVuelo := p.FechaVuelo
					actuales = append(actuales, models.DescargoTramo{
						Tipo:       models.TipoDescargoTramo(tipoTarget),
						RutaID:     p.RutaID,
						RutaPasaje: p.RutaPasaje,
						Fecha:      &tVuelo,
						Billete:    p.NumeroBillete,
						Orden:      i,
					})
					existingKeys[key] = true
				}
			}
		}
	}
	return actuales
}

func (s *DescargoDerechoService) sortTramos(tramos []models.DescargoTramo) {
	for i := range tramos {
		for j := i + 1; j < len(tramos); j++ {
			if tramos[i].Orden > tramos[j].Orden {
				tramos[i], tramos[j] = tramos[j], tramos[i]
			}
		}
	}
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
	// Mapear tramos existentes por PasajeID y Orden para evitar duplicados
	existingMap := make(map[string]bool)
	for _, det := range descargo.Tramos {
		if det.PasajeID != nil {
			key := fmt.Sprintf("%s_%d", *det.PasajeID, det.Orden)
			existingMap[key] = true
		}
	}

	modified := false
	process := func(item *models.SolicitudItem, tipoPrefix string) {
		if item == nil {
			return
		}
		for _, p := range item.Pasajes {
			if !p.IsDischargeable() {
				continue
			}

			tipo := tipoPrefix + "_ORIGINAL"

			tramosVuelo := p.GetTramosRuta()
			for i := range tramosVuelo {
				orden := p.Orden*100 + i
				key := fmt.Sprintf("%s_%d", p.ID, orden)

				if !existingMap[key] {
					tVuelo := p.FechaVuelo
					descargo.Tramos = append(descargo.Tramos, models.DescargoTramo{
						Tipo:            models.TipoDescargoTramo(tipo),
						RutaID:          p.RutaID,
						PasajeID:        &p.ID,
						SolicitudItemID: &item.ID,
						Fecha:           &tVuelo,
						Billete:         strings.ToUpper(strings.TrimSpace(p.NumeroBillete)),
						Orden:           orden,
					})
					existingMap[key] = true
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
	descargo.Estado = models.EstadoDescargoBorrador
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

		// 4. Domain Rule: Data Protection for issued segments
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

		// 5. Atomic Entity Assembly
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
			Orden:             row.Orden,
		}
		det.ID = idRow
		tramosProcesados = append(tramosProcesados, det)
	}

	descargo.Tramos = tramosProcesados
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoDerechoService) PrepareItinerarioDerecho(descargo *models.Descargo) (map[string][]dtos.TramoView, map[string][]dtos.TramoView) {
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
				Orden:           item.Orden,
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
