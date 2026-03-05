package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type RolService struct {
	repo *repositories.RolRepository
}

func NewRolService(repo *repositories.RolRepository) *RolService {
	return &RolService{
		repo: repo,
	}
}

func (s *RolService) GetAll(ctx context.Context) ([]models.Rol, error) {
	return s.repo.FindAll(ctx)
}
