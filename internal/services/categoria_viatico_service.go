package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type CategoriaViaticoService struct {
	repo *repositories.CategoriaViaticoRepository
}

func NewCategoriaViaticoService(repo *repositories.CategoriaViaticoRepository) *CategoriaViaticoService {
	return &CategoriaViaticoService{
		repo: repo,
	}
}

func (s *CategoriaViaticoService) GetAll(ctx context.Context) ([]models.CategoriaViatico, error) {
	return s.repo.FindAll(ctx)
}

func (s *CategoriaViaticoService) Create(ctx context.Context, cat *models.CategoriaViatico) error {
	return s.repo.Create(ctx, cat)
}
