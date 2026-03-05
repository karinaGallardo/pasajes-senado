package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AgenciaService struct {
	repo *repositories.AgenciaRepository
}

func NewAgenciaService(repo *repositories.AgenciaRepository) *AgenciaService {
	return &AgenciaService{
		repo: repo,
	}
}

func (s *AgenciaService) GetAllActive(ctx context.Context) ([]models.Agencia, error) {
	return s.repo.FindAllActive(ctx)
}

func (s *AgenciaService) GetAll(ctx context.Context) ([]models.Agencia, error) {
	return s.repo.FindAll(ctx)
}

func (s *AgenciaService) GetByID(ctx context.Context, id string) (*models.Agencia, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *AgenciaService) Create(ctx context.Context, agencia *models.Agencia) error {
	return s.repo.Create(ctx, agencia)
}

func (s *AgenciaService) Update(ctx context.Context, agencia *models.Agencia) error {
	return s.repo.Update(ctx, agencia)
}

func (s *AgenciaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.Update(ctx, a)
}

func (s *AgenciaService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
