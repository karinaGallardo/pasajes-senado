package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type TipoItinerarioService struct {
	repo *repositories.TipoItinerarioRepository
}

func NewTipoItinerarioService() *TipoItinerarioService {
	return &TipoItinerarioService{
		repo: repositories.NewTipoItinerarioRepository(),
	}
}

func (s *TipoItinerarioService) GetByCodigo(ctx context.Context, codigo string) (*models.TipoItinerario, error) {
	return s.repo.WithContext(ctx).FindByCodigo(codigo)
}

func (s *TipoItinerarioService) GetAll(ctx context.Context) ([]models.TipoItinerario, error) {
	return s.repo.WithContext(ctx).FindAll()
}
