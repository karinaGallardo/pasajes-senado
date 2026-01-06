package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AmbitoService struct {
	repo *repositories.AmbitoViajeRepository
}

func NewAmbitoService() *AmbitoService {
	return &AmbitoService{
		repo: repositories.NewAmbitoViajeRepository(),
	}
}

func (s *AmbitoService) GetAll() ([]models.AmbitoViaje, error) {
	return s.repo.FindAll()
}

func (s *AmbitoService) GetByCodigo(codigo string) (*models.AmbitoViaje, error) {
	return s.repo.FindByCodigo(codigo)
}
