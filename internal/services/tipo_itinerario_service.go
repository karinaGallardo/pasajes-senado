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

func (s *TipoItinerarioService) EnsureDefaults(ctx context.Context) error {
	defaults := []models.TipoItinerario{
		{Codigo: "IDA_VUELTA", Nombre: "Ida y Vuelta"},
		{Codigo: "SOLO_IDA", Nombre: "Solo Ida"},
		{Codigo: "SOLO_VUELTA", Nombre: "Solo Vuelta"},
	}

	for _, d := range defaults {
		if err := s.repo.WithContext(ctx).FirstOrCreate(&d); err != nil {
			return err
		}
	}
	return nil
}
