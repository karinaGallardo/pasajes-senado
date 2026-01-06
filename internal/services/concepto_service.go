package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type ConceptoService struct {
	repo *repositories.ConceptoViajeRepository
}

func NewConceptoService() *ConceptoService {
	return &ConceptoService{
		repo: repositories.NewConceptoViajeRepository(),
	}
}

func (s *ConceptoService) GetAll() ([]models.ConceptoViaje, error) {
	return s.repo.FindConceptos()
}

func (s *ConceptoService) GetByCodigo(codigo string) (*models.ConceptoViaje, error) {
	return s.repo.FindByCodigo(codigo)
}
