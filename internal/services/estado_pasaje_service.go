package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type EstadoPasajeService struct {
	repo *repositories.EstadoPasajeRepository
}

func NewEstadoPasajeService(repo *repositories.EstadoPasajeRepository) *EstadoPasajeService {
	return &EstadoPasajeService{
		repo: repo,
	}
}

func (s *EstadoPasajeService) GetByCodigo(ctx context.Context, codigo string) (*models.EstadoPasaje, error) {
	return s.repo.FindByCodigo(ctx, codigo)
}

func (s *EstadoPasajeService) GetAll(ctx context.Context) ([]models.EstadoPasaje, error) {
	return s.repo.FindAll(ctx)
}
