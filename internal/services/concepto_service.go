package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type ConceptoService struct {
	repo *repositories.ConceptoViajeRepository
}

func NewConceptoService(repo *repositories.ConceptoViajeRepository) *ConceptoService {
	return &ConceptoService{
		repo: repo,
	}
}

func (s *ConceptoService) GetAll(ctx context.Context) ([]models.ConceptoViaje, error) {
	return s.repo.FindConceptos(ctx)
}

func (s *ConceptoService) GetByCodigo(ctx context.Context, codigo string) (*models.ConceptoViaje, error) {
	return s.repo.FindByCodigo(ctx, codigo)
}
