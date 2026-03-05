package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AerolineaService struct {
	repo *repositories.AerolineaRepository
}

func NewAerolineaService(repo *repositories.AerolineaRepository) *AerolineaService {
	return &AerolineaService{
		repo: repo,
	}
}

func (s *AerolineaService) GetAllActive(ctx context.Context) ([]models.Aerolinea, error) {
	return s.repo.FindAllActive(ctx)
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
	return s.repo.FindAll(ctx)
}

func (s *AerolineaService) GetByID(ctx context.Context, id string) (*models.Aerolinea, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *AerolineaService) Create(ctx context.Context, aerolinea *models.Aerolinea) error {
	return s.repo.Create(ctx, aerolinea)
}

func (s *AerolineaService) Update(ctx context.Context, aerolinea *models.Aerolinea) error {
	return s.repo.Update(ctx, aerolinea)
}

func (s *AerolineaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.Update(ctx, a)
}

func (s *AerolineaService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
