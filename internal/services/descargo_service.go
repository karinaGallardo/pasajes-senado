package services

import (
	"context"
	"errors"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"time"
)

type DescargoService struct {
	repo          *repositories.DescargoRepository
	solicitudRepo *repositories.SolicitudRepository
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	solicitudRepo *repositories.SolicitudRepository,
) *DescargoService {
	return &DescargoService{
		repo:          repo,
		solicitudRepo: solicitudRepo,
	}
}

func (s *DescargoService) Create(ctx context.Context, req dtos.CreateDescargoRequest, userID string, archivoPaths []string, anexoPaths []string) (*models.Descargo, error) {
	solicitud, err := s.solicitudRepo.FindByID(ctx, req.SolicitudID)
	if err != nil {
		return nil, err
	}

	descargo := &models.Descargo{
		SolicitudID:       req.SolicitudID,
		UsuarioID:         userID,
		Codigo:            solicitud.Codigo,
		FechaPresentacion: time.Now(),
		Observaciones:     req.Observaciones,
		Estado:            "EN_REVISION",
	}
	descargo.CreatedBy = &userID

	// Si es solicitud oficial, crear detalle PV-06
	isOficial := req.NroMemorandum != "" || req.InformeActividades != "" || req.ObjetivoViaje != "" || req.ResultadosViaje != "" || req.ConclusionesRecomendaciones != "" || req.MontoDevolucion > 0

	if isOficial {
		oficial := &models.DescargoOficial{
			NroMemorandum:               req.NroMemorandum,
			ObjetivoViaje:               req.ObjetivoViaje,
			TipoTransporte:              req.TipoTransporte,
			PlacaVehiculo:               req.PlacaVehiculo,
			InformeActividades:          req.InformeActividades,
			ResultadosViaje:             req.ResultadosViaje,
			ConclusionesRecomendaciones: req.ConclusionesRecomendaciones,
			MontoDevolucion:             req.MontoDevolucion,
			NroBoletaDeposito:           req.NroBoletaDeposito,
			DirigidoA:                   req.DirigidoA,
		}

		// Mapear Anexos
		var anexos []models.AnexoDescargo
		for i, path := range anexoPaths {
			if path != "" {
				anexos = append(anexos, models.AnexoDescargo{
					Archivo: path,
					Orden:   i,
				})
			}
		}
		oficial.Anexos = anexos
		descargo.Oficial = oficial
	}

	// Mapear Detalles de Itinerario
	devolucionMap := make(map[string]bool)
	for _, idx := range req.ItinDevolucion {
		devolucionMap[idx] = true
	}

	modificacionMap := make(map[string]bool)
	for _, idx := range req.ItinModificacion {
		modificacionMap[idx] = true
	}

	var itinDetalles []models.DetalleItinerarioDescargo
	for i := range req.ItinTipo {
		if i < len(req.ItinRuta) && req.ItinRuta[i] != "" {
			var fecha *time.Time
			if i < len(req.ItinFecha) && req.ItinFecha[i] != "" {
				t := utils.ParseDate("2006-01-02", req.ItinFecha[i])
				fecha = &t
			}

			boleto := ""
			if i < len(req.ItinBoleto) {
				boleto = req.ItinBoleto[i]
			}

			paseNumero := ""
			if i < len(req.ItinPaseNumero) {
				paseNumero = req.ItinPaseNumero[i]
			}

			archivoPath := ""
			if i < len(archivoPaths) {
				archivoPath = archivoPaths[i]
			}

			esDevo := false
			esMod := false
			if i < len(req.ItinIndex) {
				esDevo = devolucionMap[req.ItinIndex[i]]
				esMod = modificacionMap[req.ItinIndex[i]]
			}

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				EsModificacion:    esMod,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}
	descargo.DetallesItinerario = itinDetalles

	if err := s.repo.Create(ctx, descargo); err != nil {
		return nil, err
	}

	return descargo, nil
}

func (s *DescargoService) GetBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.FindBySolicitudID(ctx, solicitudID)
}

func (s *DescargoService) GetByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *DescargoService) GetAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.FindAll(ctx)
}

func (s *DescargoService) GetPaginated(ctx context.Context, page, limit int, searchTerm string) (*repositories.PaginatedDescargos, error) {
	return s.repo.FindPaginated(ctx, page, limit, searchTerm)
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoService) UpdateFull(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, archivoPaths []string, anexoPaths []string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	descargo.Observaciones = req.Observaciones
	descargo.UpdatedBy = &userID

	// Mapear Detalle Oficial
	isOficialUpdate := req.NroMemorandum != "" || req.InformeActividades != "" || req.ObjetivoViaje != "" || req.ResultadosViaje != "" || req.ConclusionesRecomendaciones != "" || req.MontoDevolucion > 0

	if isOficialUpdate {
		if descargo.Oficial == nil {
			descargo.Oficial = &models.DescargoOficial{DescargoID: id}
		}
		descargo.Oficial.NroMemorandum = req.NroMemorandum
		descargo.Oficial.ObjetivoViaje = req.ObjetivoViaje
		descargo.Oficial.TipoTransporte = req.TipoTransporte
		descargo.Oficial.PlacaVehiculo = req.PlacaVehiculo
		descargo.Oficial.InformeActividades = req.InformeActividades
		descargo.Oficial.ResultadosViaje = req.ResultadosViaje
		descargo.Oficial.ConclusionesRecomendaciones = req.ConclusionesRecomendaciones
		descargo.Oficial.MontoDevolucion = req.MontoDevolucion
		descargo.Oficial.NroBoletaDeposito = req.NroBoletaDeposito
		descargo.Oficial.DirigidoA = req.DirigidoA

		// Explicitly save the official detail to ensure columns are updated correctly
		if err := s.repo.UpdateOficial(ctx, descargo.Oficial); err != nil {
			return err
		}

		// Mapear Anexos
		if len(anexoPaths) > 0 {
			var anexos []models.AnexoDescargo
			for i, path := range anexoPaths {
				if path != "" {
					anexos = append(anexos, models.AnexoDescargo{
						DescargoOficialID: descargo.Oficial.ID,
						Archivo:           path,
						Orden:             i,
					})
				}
			}
			// Clear existing ones using oficial ID if exists
			if descargo.Oficial.ID != "" {
				if err := s.repo.ClearAnexos(ctx, descargo.Oficial.ID); err != nil {
					return err
				}
			}
			descargo.Oficial.Anexos = anexos
		}
	}

	// Mapear Detalles de Itinerario
	devolucionMap := make(map[string]bool)
	for _, idx := range req.ItinDevolucion {
		devolucionMap[idx] = true
	}

	modificacionMap := make(map[string]bool)
	for _, idx := range req.ItinModificacion {
		modificacionMap[idx] = true
	}

	var itinDetalles []models.DetalleItinerarioDescargo
	for i := range req.ItinTipo {
		if i < len(req.ItinRuta) && req.ItinRuta[i] != "" {
			var fecha *time.Time
			if i < len(req.ItinFecha) && req.ItinFecha[i] != "" {
				t := utils.ParseDate("2006-01-02", req.ItinFecha[i])
				fecha = &t
			}

			boleto := ""
			if i < len(req.ItinBoleto) {
				boleto = req.ItinBoleto[i]
			}

			paseNumero := ""
			if i < len(req.ItinPaseNumero) {
				paseNumero = req.ItinPaseNumero[i]
			}

			archivoPath := ""
			if i < len(archivoPaths) {
				archivoPath = archivoPaths[i]
			}

			esDevo := false
			esMod := false
			if i < len(req.ItinIndex) {
				esDevo = devolucionMap[req.ItinIndex[i]]
				esMod = modificacionMap[req.ItinIndex[i]]
			}

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				DescargoID:        id,
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				EsModificacion:    esMod,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}

	if err := s.repo.ClearDetalles(ctx, id); err != nil {
		return err
	}

	descargo.DetallesItinerario = itinDetalles
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoService) RevertApproval(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != "APROBADO" {
		return errors.New("el descargo no está en estado APROBADO")
	}

	descargo.Estado = "EN_REVISION"
	descargo.UpdatedBy = &userID

	return s.repo.Update(ctx, descargo)
}
