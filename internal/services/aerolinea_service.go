package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AerolineaService struct {
	repo *repositories.AerolineaRepository
}

func NewAerolineaService() *AerolineaService {
	return &AerolineaService{
		repo: repositories.NewAerolineaRepository(),
	}
}

func (s *AerolineaService) GetAllActive(ctx context.Context) ([]models.Aerolinea, error) {
	return s.repo.WithContext(ctx).FindAllActive()
}

func (s *AerolineaService) GetActiveNames(ctx context.Context) ([]string, error) {
	aereos, err := s.GetAllActive(ctx)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, a := range aereos {
		names = append(names, a.Nombre)
	}
	return names, nil
}

func (s *AerolineaService) GetAll(ctx context.Context) ([]models.Aerolinea, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *AerolineaService) GetByID(ctx context.Context, id string) (*models.Aerolinea, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *AerolineaService) Create(ctx context.Context, aerolinea *models.Aerolinea) error {
	return s.repo.WithContext(ctx).Create(aerolinea)
}

func (s *AerolineaService) Update(ctx context.Context, aerolinea *models.Aerolinea) error {
	return s.repo.WithContext(ctx).Update(aerolinea)
}

func (s *AerolineaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.WithContext(ctx).Update(a)
}

func (s *AerolineaService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).Delete(id)
}
