package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type CreditoPasajeService struct {
	repo *repositories.CreditoPasajeRepository
}

func NewCreditoPasajeService(repo *repositories.CreditoPasajeRepository) *CreditoPasajeService {
	return &CreditoPasajeService{repo: repo}
}

func (s *CreditoPasajeService) GetByUsuarioID(ctx context.Context, usuarioID string) ([]models.CreditoPasaje, error) {
	return s.repo.FindByUsuarioID(ctx, usuarioID)
}

func (s *CreditoPasajeService) GetByID(ctx context.Context, id string) (*models.CreditoPasaje, error) {
	return s.repo.FindByID(ctx, id)
}
