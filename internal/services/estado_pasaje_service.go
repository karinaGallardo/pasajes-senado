package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type EstadoPasajeService struct {
	repo *repositories.EstadoPasajeRepository
}

func NewEstadoPasajeService() *EstadoPasajeService {
	return &EstadoPasajeService{
		repo: repositories.NewEstadoPasajeRepository(),
	}
}

func (s *EstadoPasajeService) GetByCodigo(ctx context.Context, codigo string) (*models.EstadoPasaje, error) {
	return s.repo.WithContext(ctx).FindByCodigo(codigo)
}

func (s *EstadoPasajeService) GetAll(ctx context.Context) ([]models.EstadoPasaje, error) {
	return s.repo.WithContext(ctx).FindAll()
}
