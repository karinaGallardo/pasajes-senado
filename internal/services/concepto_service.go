package services

import (
	"context"
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

func (s *ConceptoService) GetAll(ctx context.Context) ([]models.ConceptoViaje, error) {
	return s.repo.WithContext(ctx).FindConceptos()
}

func (s *ConceptoService) GetByCodigo(ctx context.Context, codigo string) (*models.ConceptoViaje, error) {
	return s.repo.WithContext(ctx).FindByCodigo(codigo)
}
