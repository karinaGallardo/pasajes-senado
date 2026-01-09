package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AmbitoService struct {
	repo *repositories.AmbitoViajeRepository
}

func NewAmbitoService() *AmbitoService {
	return &AmbitoService{
		repo: repositories.NewAmbitoViajeRepository(),
	}
}

func (s *AmbitoService) GetAll(ctx context.Context) ([]models.AmbitoViaje, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *AmbitoService) GetByCodigo(ctx context.Context, codigo string) (*models.AmbitoViaje, error) {
	return s.repo.WithContext(ctx).FindByCodigo(codigo)
}
