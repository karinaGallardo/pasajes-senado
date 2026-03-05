package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type TipoSolicitudService struct {
	repo *repositories.TipoSolicitudRepository
}

func NewTipoSolicitudService(repo *repositories.TipoSolicitudRepository) *TipoSolicitudService {
	return &TipoSolicitudService{
		repo: repo,
	}
}

func (s *TipoSolicitudService) GetByConcepto(ctx context.Context, conceptoCodigo string) ([]models.TipoSolicitud, error) {
	return s.repo.FindByConceptoCodigo(ctx, conceptoCodigo)
}

func (s *TipoSolicitudService) GetAmbitosByTipo(ctx context.Context, tipoCodigo string) ([]models.AmbitoViaje, error) {
	return s.repo.FindAmbitosByTipoCodigo(ctx, tipoCodigo)
}

func (s *TipoSolicitudService) GetByID(ctx context.Context, id string) (*models.TipoSolicitud, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *TipoSolicitudService) GetAll(ctx context.Context) ([]models.TipoSolicitud, error) {
	return s.repo.FindAll(ctx)
}

func (s *TipoSolicitudService) GetByCodigo(ctx context.Context, codigo string) (*models.TipoSolicitud, error) {
	return s.repo.FindByCodigo(ctx, codigo)
}

func (s *TipoSolicitudService) GetByCodigoAndAmbito(ctx context.Context, tipoCodigo, ambitoCodigo string) (*models.TipoSolicitud, *models.AmbitoViaje, error) {
	return s.repo.FindByCodigoAndAmbito(ctx, tipoCodigo, ambitoCodigo)
}
