package services

import (
	"context"
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

func NewDescargoService() *DescargoService {
	return &DescargoService{
		repo:          repositories.NewDescargoRepository(),
		solicitudRepo: repositories.NewSolicitudRepository(),
	}
}

func (s *DescargoService) Create(ctx context.Context, req dtos.CreateDescargoRequest, userID string, archivoPaths []string) (*models.Descargo, error) {
	solicitud, err := s.solicitudRepo.WithContext(ctx).FindByID(req.SolicitudID)
	if err != nil {
		return nil, err
	}

	descargo := &models.Descargo{
		SolicitudID:        req.SolicitudID,
		UsuarioID:          userID,
		Codigo:             solicitud.Codigo,
		FechaPresentacion:  time.Now(),
		InformeActividades: req.InformeActividades,
		Observaciones:      req.Observaciones,
		Estado:             "EN_REVISION",
	}
	descargo.CreatedBy = &userID

	// Mapear Detalles de Itinerario
	devolucionMap := make(map[string]bool)
	for _, idx := range req.ItinDevolucion {
		devolucionMap[idx] = true
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
			if i < len(req.ItinIndex) {
				esDevo = devolucionMap[req.ItinIndex[i]]
			}

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}
	descargo.DetallesItinerario = itinDetalles

	if err := s.repo.WithContext(ctx).Create(descargo); err != nil {
		return nil, err
	}

	return descargo, nil
}

func (s *DescargoService) GetBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *DescargoService) GetByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *DescargoService) GetAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.WithContext(ctx).Update(descargo)
}

func (s *DescargoService) UpdateFull(ctx context.Context, id string, req dtos.CreateDescargoRequest, userID string, archivoPaths []string) error {
	descargo, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}

	descargo.InformeActividades = req.InformeActividades
	descargo.Observaciones = req.Observaciones
	descargo.UpdatedBy = &userID

	// Mapear Detalles de Itinerario
	devolucionMap := make(map[string]bool)
	for _, idx := range req.ItinDevolucion {
		devolucionMap[idx] = true
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
			if i < len(req.ItinIndex) {
				esDevo = devolucionMap[req.ItinIndex[i]]
			}

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				DescargoID:        id,
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
				EsDevolucion:      esDevo,
				NumeroPaseAbordo:  paseNumero,
				ArchivoPaseAbordo: archivoPath,
				Orden:             i,
			})
		}
	}

	// Iniciar Transacción? The repo doesn't seem to expose it easily, but I can just do it.
	// For now, I'll clear and save.
	if err := s.repo.WithContext(ctx).ClearDetalles(id); err != nil {
		return err
	}

	descargo.DetallesItinerario = itinDetalles
	return s.repo.WithContext(ctx).Update(descargo)
}
