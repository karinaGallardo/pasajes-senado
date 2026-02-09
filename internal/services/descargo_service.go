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
	repo *repositories.DescargoRepository
}

func NewDescargoService() *DescargoService {
	return &DescargoService{
		repo: repositories.NewDescargoRepository(),
	}
}

func (s *DescargoService) Create(ctx context.Context, req dtos.CreateDescargoRequest, userID string, archivoPaths []string) (*models.Descargo, error) {
	codigo := utils.GenerateYearlyCode("D", 6)

	descargo := &models.Descargo{
		SolicitudID:        req.SolicitudID,
		UsuarioID:          userID,
		Codigo:             codigo,
		FechaPresentacion:  time.Now(),
		InformeActividades: req.InformeActividades,
		Observaciones:      req.Observaciones,
		Estado:             "EN_REVISION",
	}
	descargo.CreatedBy = &userID

	// Mapear Detalles de Itinerario
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

			itinDetalles = append(itinDetalles, models.DetalleItinerarioDescargo{
				Tipo:              models.TipoDetalleItinerario(req.ItinTipo[i]),
				Ruta:              req.ItinRuta[i],
				Fecha:             fecha,
				Boleto:            boleto,
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
