package services

import (
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

func (s *TipoItinerarioService) GetByCodigo(codigo string) (*models.TipoItinerario, error) {
	return s.repo.FindByCodigo(codigo)
}

func (s *TipoItinerarioService) GetAll() ([]models.TipoItinerario, error) {
	return s.repo.FindAll()
}
