package services

import (
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

func (s *EstadoPasajeService) GetByCodigo(codigo string) (*models.EstadoPasaje, error) {
	return s.repo.FindByCodigo(codigo)
}

func (s *EstadoPasajeService) GetAll() ([]models.EstadoPasaje, error) {
	return s.repo.FindAll()
}
