package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DestinoService struct {
	repo *repositories.DestinoRepository
}

func NewDestinoService() *DestinoService {
	return &DestinoService{
		repo: repositories.NewDestinoRepository(),
	}
}

func (s *DestinoService) GetAll() ([]models.Destino, error) {
	return s.repo.FindAll()
}

func (s *DestinoService) GetByAmbito(ambito string) ([]models.Destino, error) {
	return s.repo.FindByAmbito(ambito)
}

func (s *DestinoService) GetByIATA(iata string) (*models.Destino, error) {
	return s.repo.FindByIATA(iata)
}
