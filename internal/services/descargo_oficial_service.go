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

func (s *DescargoOficialService) UpdateOficial(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, archivoPaths []string, anexoPaths []string) error {
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
		for i, path := range anexoPaths {
			if path != "" {
				anexos = append(anexos, models.AnexoDescargo{DescargoOficialID: descargo.Oficial.ID, Archivo: path, Orden: i})
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

	// 4. Itinerary Connection Data
	devoMap := utils.ToMap(req.ItinDevolucion)
	modMap := utils.ToMap(req.ItinModificacion)
	existingMap := make(map[string]models.DetalleItinerarioDescargo)
	for _, d := range descargo.DetallesItinerario {
		existingMap[d.ID] = d
	}

	var itinDetalles []models.DetalleItinerarioDescargo
	for i, tipoRaw := range req.ItinTipo {
		// Mandatory field check
		rutaIDRaw := utils.GetIdx(req.ItinRutaID, i)
		if rutaIDRaw == "" {
			continue
		}

		rawID := utils.GetIdx(req.ItinID, i)
		idRow := strings.TrimSpace(rawID)
		if strings.HasPrefix(idRow, "new_") || idRow == "" {
			idRow = ""
		}

		fecha := utils.ParseDatePtr("2006-01-02", utils.GetIdx(req.ItinFecha, i))
		boleto := strings.ToUpper(strings.TrimSpace(utils.GetIdx(req.ItinBoleto, i)))
		paseNum := utils.GetIdx(req.ItinPaseNumero, i)
		archivo := utils.GetIdx(archivoPaths, i)
		moneda := utils.GetIdxOrDefault(req.ItinMoneda, i, "Bs.")
		montoDevo, _ := strconv.ParseFloat(utils.GetIdx(req.ItinMontoDevolucion, i), 64)
		orden, _ := strconv.Atoi(utils.GetIdx(req.ItinOrden, i))

		rutaID := utils.NilIfEmpty(rutaIDRaw)
		pasajeID := utils.NilIfEmpty(utils.GetIdx(req.ItinPasajeID, i))
		solItemID := utils.NilIfEmpty(utils.GetIdx(req.ItinSolicitudItemID, i))

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

		tipoRow := models.TipoDetalleItinerario(tipoRaw)
		if idRow != "" {
			if original, ok := existingMap[idRow]; ok && original.PasajeID != nil {
				tipoRow = original.Tipo
				rutaID = original.RutaID
				fecha = original.Fecha
				boleto = original.Boleto
				pasajeID = original.PasajeID
				solItemID = original.SolicitudItemID
			}
		}

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

func (s *DescargoOficialService) PrepareItinerarioOficial(descargo *models.Descargo) (map[string][]dtos.ConnectionView, map[string][]dtos.ConnectionView) {
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
