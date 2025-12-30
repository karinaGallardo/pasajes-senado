package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type DescargoService struct {
	repo *repositories.DescargoRepository
}

func NewDescargoService() *DescargoService {
	return &DescargoService{
		repo: repositories.NewDescargoRepository(),
	}
}

func (s *DescargoService) Create(descargo *models.Descargo) error {
	return s.repo.Create(descargo)
}

func (s *DescargoService) FindBySolicitudID(solicitudID string) (*models.Descargo, error) {
	return s.repo.FindBySolicitudID(solicitudID)
}

func (s *DescargoService) FindByID(id string) (*models.Descargo, error) {
	return s.repo.FindByID(id)
}

func (s *DescargoService) FindAll() ([]models.Descargo, error) {
	return s.repo.FindAll()
}

func (s *DescargoService) Update(descargo *models.Descargo) error {
	return s.repo.Update(descargo)
}
