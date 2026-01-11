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

func (s *AgenciaService) FindByID(ctx context.Context, id string) (*models.Agencia, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *AgenciaService) Create(ctx context.Context, nombre, telefono string, estado bool) (*models.Agencia, error) {
	agencia := &models.Agencia{Nombre: nombre, Telefono: telefono, Estado: estado}
	err := s.repo.WithContext(ctx).Create(agencia)
	return agencia, err
}

func (s *AgenciaService) Update(ctx context.Context, id string, nombre, telefono string, estado bool) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Nombre = nombre
	a.Telefono = telefono
	a.Estado = estado
	return s.repo.WithContext(ctx).Save(a)
}

func (s *AgenciaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.WithContext(ctx).Save(a)
}

func (s *AgenciaService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).Delete(id)
}
