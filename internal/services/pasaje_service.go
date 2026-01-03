package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type PasajeService struct {
	repo *repositories.PasajeRepository
}

func NewPasajeService() *PasajeService {
	return &PasajeService{
		repo: repositories.NewPasajeRepository(),
	}
}

func (s *PasajeService) Create(pasaje *models.Pasaje) error {
	return s.repo.Create(pasaje)
}

func (s *PasajeService) FindBySolicitudID(solicitudID string) ([]models.Pasaje, error) {
	return s.repo.FindBySolicitudID(solicitudID)
}

func (s *PasajeService) Delete(id uint) error {
	return s.repo.Delete(id)
}
