package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DestinoService struct {
	repo *repositories.DestinoRepository
}

func NewDestinoService(repo *repositories.DestinoRepository) *DestinoService {
	return &DestinoService{
		repo: repo,
	}
}

func (s *DestinoService) GetAll(ctx context.Context) ([]models.Destino, error) {
	return s.repo.FindAll(ctx)
}

func (s *DestinoService) GetByAmbito(ctx context.Context, ambito string) ([]models.Destino, error) {
	return s.repo.FindByAmbito(ctx, ambito)
}

func (s *DestinoService) GetByIATA(ctx context.Context, iata string) (*models.Destino, error) {
	return s.repo.FindByIATA(ctx, iata)
}

func (s *DestinoService) Search(ctx context.Context, query string, ambito string, page, pageSize int) ([]models.Destino, int64, error) {
	return s.repo.Search(ctx, query, ambito, page, pageSize)
}

func (s *DestinoService) Create(ctx context.Context, d *models.Destino) error {
	existing, _ := s.repo.FindByIATA(ctx, d.IATA)
	if existing != nil {
		return fmt.Errorf("el código IATA '%s' ya está registrado", d.IATA)
	}
	return s.repo.Create(ctx, d)
}

func (s *DestinoService) Update(ctx context.Context, d *models.Destino) error {
	return s.repo.Update(ctx, d)
}

func (s *DestinoService) Toggle(ctx context.Context, iata string) error {
	d, err := s.repo.FindByIATA(ctx, iata)
	if err != nil {
		return err
	}
	d.Estado = !d.Estado
	return s.repo.Update(ctx, d)
}

func (s *DestinoService) Delete(ctx context.Context, iata string) error {
	return s.repo.Delete(ctx, iata)
}
