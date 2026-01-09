package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DescargoService struct {
	repo *repositories.DescargoRepository
}

func NewDescargoService() *DescargoService {
	return &DescargoService{
		repo: repositories.NewDescargoRepository(),
	}
}

func (s *DescargoService) Create(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.WithContext(ctx).Create(descargo)
}

func (s *DescargoService) FindBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *DescargoService) FindByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *DescargoService) FindAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.WithContext(ctx).Update(descargo)
}
