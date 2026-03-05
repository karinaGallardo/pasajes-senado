package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AmbitoService struct {
	repo *repositories.AmbitoViajeRepository
}

func NewAmbitoService(repo *repositories.AmbitoViajeRepository) *AmbitoService {
	return &AmbitoService{
		repo: repo,
	}
}

func (s *AmbitoService) GetAll(ctx context.Context) ([]models.AmbitoViaje, error) {
	return s.repo.FindAll(ctx)
}

func (s *AmbitoService) GetByCodigo(ctx context.Context, codigo string) (*models.AmbitoViaje, error) {
	return s.repo.FindByCodigo(ctx, codigo)
}
