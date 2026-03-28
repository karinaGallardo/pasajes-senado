package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DestinoService struct {
	repo *repositories.DestinoRepository
}

func NewDestinoService(repo *repositories.DestinoRepository) *DestinoService {
	return &DestinoService{
		repo: repo,
	}
}

func (s *DestinoService) GetAll(ctx context.Context) ([]models.Destino, error) {
	return s.repo.FindAll(ctx)
}

func (s *DestinoService) GetByAmbito(ctx context.Context, ambito string) ([]models.Destino, error) {
	return s.repo.FindByAmbito(ctx, ambito)
}

func (s *DestinoService) GetByIATA(ctx context.Context, iata string) (*models.Destino, error) {
	return s.repo.FindByIATA(ctx, iata)
}

func (s *DestinoService) Search(ctx context.Context, query string) ([]models.Destino, error) {
	return s.repo.Search(ctx, query)
}
