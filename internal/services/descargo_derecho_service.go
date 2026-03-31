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
		UsuarioID:         userID,
		Codigo:            solicitud.Codigo,
		FechaPresentacion: time.Now(),
		Estado:            models.EstadoDescargoBorrador,
	}
	descargo.CreatedBy = &userID

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
	// Mapear segmentos existentes por PasajeID y Orden para evitar duplicados
	existingMap := make(map[string]bool)
	for _, det := range descargo.DetallesItinerario {
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
			st := p.GetEstadoCodigo()
			if st != "EMITIDO" && st != "USADO" {
				continue
			}

			tipo := tipoPrefix + "_ORIGINAL"
			if p.PasajeAnteriorID != nil {
				tipo = tipoPrefix + "_REPRO"
			}

			segments := p.GetRutaSegments()
			for i := range segments {
				orden := p.Orden*100 + i
				key := fmt.Sprintf("%s_%d", p.ID, orden)

				if !existingMap[key] {
					tVuelo := p.FechaVuelo
					descargo.DetallesItinerario = append(descargo.DetallesItinerario, models.DetalleItinerarioDescargo{
						Tipo:            models.TipoDetalleItinerario(tipo),
						RutaID:          p.RutaID,
						PasajeID:        &p.ID,
						SolicitudItemID: &item.ID,
						Fecha:           &tVuelo,
						Boleto:          strings.ToUpper(strings.TrimSpace(p.NumeroBoleto)),
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

func (s *DescargoDerechoService) UpdateDerecho(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, archivoPaths []string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		return fmt.Errorf("el descargo no se puede editar en su estado actual (%s)", descargo.Estado)
	}

	// 1. Basic Metadata
	descargo.Observaciones = req.Observaciones
	descargo.Estado = models.EstadoDescargoBorrador
	descargo.UpdatedBy = &userID

	// 2. Index Maps for Quick Lookup
	devoMap := utils.ToMap(req.ItinDevolucion)
	modMap := utils.ToMap(req.ItinModificacion)
	existingMap := make(map[string]models.DetalleItinerarioDescargo)
	for _, d := range descargo.DetallesItinerario {
		existingMap[d.ID] = d
	}

	// 3. Process Itinerary Rows
	var itinDetalles []models.DetalleItinerarioDescargo
	for i, tipoRaw := range req.ItinTipo {
		// Mandatory field check
		rutaIDRaw := utils.GetIdx(req.ItinRutaID, i)
		if rutaIDRaw == "" {
			continue
		}

		// Row Data Extraction
		rawID := utils.GetIdx(req.ItinID, i)
		idRow := strings.TrimSpace(rawID)
		if strings.HasPrefix(idRow, "new_") || idRow == "" {
			idRow = ""
		}

		// Basic fields
		fecha := utils.ParseDatePtr("2006-01-02", utils.GetIdx(req.ItinFecha, i))
		boleto := strings.ToUpper(strings.TrimSpace(utils.GetIdx(req.ItinBoleto, i)))
		paseNum := utils.GetIdx(req.ItinPaseNumero, i)
		archivo := utils.GetIdx(archivoPaths, i)
		moneda := utils.GetIdxOrDefault(req.ItinMoneda, i, "Bs.")
		montoDevo, _ := strconv.ParseFloat(utils.GetIdx(req.ItinMontoDevolucion, i), 64)
		orden, _ := strconv.Atoi(utils.GetIdx(req.ItinOrden, i))

		// Relationships
		rutaID := utils.NilIfEmpty(rutaIDRaw)
		pasajeID := utils.NilIfEmpty(utils.GetIdx(req.ItinPasajeID, i))
		solItemID := utils.NilIfEmpty(utils.GetIdx(req.ItinSolicitudItemID, i))

		// Checkbox Status
		esDevo := devoMap[idRow]
		if idRow == "" {
			esDevo = devoMap[rawID]
		}
		esMod := modMap[idRow]
		if idRow == "" {
			esMod = modMap[rawID]
		}
		if esMod {
			esDevo = false
		}

		// 4. Data Protection Rule: Atomic connection protection
		tipoRow := models.TipoDetalleItinerario(tipoRaw)
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				// Fields from a pre-issued ticket segment cannot be modified via Discharge form
				tipoRow = original.Tipo
				rutaID = original.RutaID
				fecha = original.Fecha
				boleto = original.Boleto
				pasajeID = original.PasajeID
				solItemID = original.SolicitudItemID
			}
		}

		// 5. Assemble Entity
		det := models.DetalleItinerarioDescargo{
			DescargoID:        id,
			Tipo:              tipoRow,
			RutaID:            rutaID,
			PasajeID:          pasajeID,
			SolicitudItemID:   solItemID,
			Fecha:             fecha,
			Boleto:            boleto,
			NumeroPaseAbordo:  paseNum,
			ArchivoPaseAbordo: archivo,
			EsDevolucion:      esDevo,
			EsModificacion:    esMod,
			MontoDevolucion:   montoDevo,
			Moneda:            moneda,
			Orden:             orden,
		}
		det.ID = idRow
		itinDetalles = append(itinDetalles, det)
	}

	descargo.DetallesItinerario = itinDetalles
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoDerechoService) PrepareItinerarioDerecho(descargo *models.Descargo) (map[string][]dtos.ConnectionView, map[string][]dtos.ConnectionView) {
	pasajesOriginales := make(map[string][]dtos.ConnectionView)
	pasajesReprogramados := make(map[string][]dtos.ConnectionView)

	itemsByType := make(map[string][]models.DetalleItinerarioDescargo)

	for _, item := range descargo.DetallesItinerario {
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

			cv := dtos.ConnectionView{
				ID:              item.ID,
				Tipo:            string(item.Tipo),
				Ruta:            rv,
				RutaID:          utils.DerefString(item.RutaID),
				Fecha:           dateStr,
				Boleto:          item.Boleto,
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
