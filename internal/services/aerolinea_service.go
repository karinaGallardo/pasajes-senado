package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type AerolineaService struct {
	repo *repositories.AerolineaRepository
}

func NewAerolineaService() *AerolineaService {
	return &AerolineaService{
		repo: repositories.NewAerolineaRepository(),
	}
}

func (s *AerolineaService) GetAllActive() ([]models.Aerolinea, error) {
	return s.repo.FindAllActive()
}

func (s *AerolineaService) GetAll() ([]models.Aerolinea, error) {
	return s.repo.FindAll()
}

func (s *AerolineaService) Create(nombre string) (*models.Aerolinea, error) {
	aereo := &models.Aerolinea{Nombre: nombre, Estado: true}
	err := s.repo.Create(aereo)
	return aereo, err
}

func (s *AerolineaService) Toggle(id string) error {
	a, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.Save(a)
}
