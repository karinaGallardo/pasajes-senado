package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AgenciaService struct {
	repo *repositories.AgenciaRepository
}

func NewAgenciaService() *AgenciaService {
	return &AgenciaService{
		repo: repositories.NewAgenciaRepository(),
	}
}

func (s *AgenciaService) GetAllActive(ctx context.Context) ([]models.Agencia, error) {
	return s.repo.WithContext(ctx).FindAllActive()
}

func (s *AgenciaService) GetAll(ctx context.Context) ([]models.Agencia, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *AgenciaService) GetByID(ctx context.Context, id string) (*models.Agencia, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *AgenciaService) Create(ctx context.Context, agencia *models.Agencia) error {
	return s.repo.WithContext(ctx).Create(agencia)
}

func (s *AgenciaService) Update(ctx context.Context, agencia *models.Agencia) error {
	return s.repo.WithContext(ctx).Update(agencia)
}

func (s *AgenciaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.WithContext(ctx).Update(a)
}

func (s *AgenciaService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).Delete(id)
}
