package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type TipoItinerarioService struct {
	repo *repositories.TipoItinerarioRepository
}

func NewTipoItinerarioService(repo *repositories.TipoItinerarioRepository) *TipoItinerarioService {
	return &TipoItinerarioService{
		repo: repo,
	}
}

func (s *TipoItinerarioService) GetByCodigo(ctx context.Context, codigo string) (*models.TipoItinerario, error) {
	return s.repo.FindByCodigo(ctx, codigo)
}

func (s *TipoItinerarioService) GetAll(ctx context.Context) ([]models.TipoItinerario, error) {
	return s.repo.FindAll(ctx)
}

func (s *TipoItinerarioService) EnsureDefaults(ctx context.Context) error {
	defaults := []models.TipoItinerario{
		{Codigo: "IDA_VUELTA", Nombre: "Ida y Vuelta"},
		{Codigo: "SOLO_IDA", Nombre: "Solo Ida"},
		{Codigo: "SOLO_VUELTA", Nombre: "Solo Vuelta"},
	}

	for _, d := range defaults {
		if err := s.repo.FirstOrCreate(ctx, &d); err != nil {
			return err
		}
	}
	return nil
}
