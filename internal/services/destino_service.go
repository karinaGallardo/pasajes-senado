package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DestinoService struct {
	repo *repositories.DestinoRepository
}

func NewDestinoService() *DestinoService {
	return &DestinoService{
		repo: repositories.NewDestinoRepository(),
	}
}

func (s *DestinoService) GetAll(ctx context.Context) ([]models.Destino, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *DestinoService) GetByAmbito(ctx context.Context, ambito string) ([]models.Destino, error) {
	return s.repo.WithContext(ctx).FindByAmbito(ambito)
}

func (s *DestinoService) GetByIATA(ctx context.Context, iata string) (*models.Destino, error) {
	return s.repo.WithContext(ctx).FindByIATA(iata)
}
