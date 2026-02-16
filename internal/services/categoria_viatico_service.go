package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type CategoriaViaticoService struct {
	repo *repositories.CategoriaViaticoRepository
}

func NewCategoriaViaticoService() *CategoriaViaticoService {
	return &CategoriaViaticoService{
		repo: repositories.NewCategoriaViaticoRepository(),
	}
}

func (s *CategoriaViaticoService) GetAll(ctx context.Context) ([]models.CategoriaViatico, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *CategoriaViaticoService) Create(ctx context.Context, cat *models.CategoriaViatico) error {
	return s.repo.WithContext(ctx).Create(cat)
}
