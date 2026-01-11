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

func (s *AerolineaService) FindByID(ctx context.Context, id string) (*models.Aerolinea, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *AerolineaService) Create(ctx context.Context, nombre string, estado bool) (*models.Aerolinea, error) {
	aereo := &models.Aerolinea{Nombre: nombre, Estado: estado}
	err := s.repo.WithContext(ctx).Create(aereo)
	return aereo, err
}

func (s *AerolineaService) Update(ctx context.Context, id string, nombre string, estado bool) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Nombre = nombre
	a.Estado = estado
	return s.repo.WithContext(ctx).Save(a)
}

func (s *AerolineaService) Toggle(ctx context.Context, id string) error {
	a, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}
	a.Estado = !a.Estado
	return s.repo.WithContext(ctx).Save(a)
}

func (s *AerolineaService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).Delete(id)
}
