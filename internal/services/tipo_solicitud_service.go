package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type TipoSolicitudService struct {
	repo *repositories.TipoSolicitudRepository
}

func NewTipoSolicitudService() *TipoSolicitudService {
	return &TipoSolicitudService{
		repo: repositories.NewTipoSolicitudRepository(),
	}
}

func (s *TipoSolicitudService) GetByConcepto(ctx context.Context, conceptoCodigo string) ([]models.TipoSolicitud, error) {
	return s.repo.WithContext(ctx).FindByConceptoCodigo(conceptoCodigo)
}

func (s *TipoSolicitudService) GetAmbitosByTipo(ctx context.Context, tipoCodigo string) ([]models.AmbitoViaje, error) {
	return s.repo.WithContext(ctx).FindAmbitosByTipoCodigo(tipoCodigo)
}

func (s *TipoSolicitudService) GetByID(ctx context.Context, id string) (*models.TipoSolicitud, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *TipoSolicitudService) GetAll(ctx context.Context) ([]models.TipoSolicitud, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *TipoSolicitudService) GetByCodigo(ctx context.Context, codigo string) (*models.TipoSolicitud, error) {
	return s.repo.WithContext(ctx).FindByCodigo(codigo)
}

func (s *TipoSolicitudService) GetByCodigoAndAmbito(ctx context.Context, tipoCodigo, ambitoCodigo string) (*models.TipoSolicitud, *models.AmbitoViaje, error) {
	return s.repo.WithContext(ctx).FindByCodigoAndAmbito(tipoCodigo, ambitoCodigo)
}
