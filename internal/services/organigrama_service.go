package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type OrganigramaService struct {
	cargoRepo   *repositories.CargoRepository
	oficinaRepo *repositories.OficinaRepository
}

func NewOrganigramaService(cargoRepo *repositories.CargoRepository, oficinaRepo *repositories.OficinaRepository) *OrganigramaService {
	return &OrganigramaService{
		cargoRepo:   cargoRepo,
		oficinaRepo: oficinaRepo,
	}
}

func (s *OrganigramaService) GetAllCargos(ctx context.Context) ([]models.Cargo, error) {
	return s.cargoRepo.FindAll(ctx)
}

func (s *OrganigramaService) CreateCargo(ctx context.Context, cargo *models.Cargo) error {
	return s.cargoRepo.Create(ctx, cargo)
}

func (s *OrganigramaService) DeleteCargo(ctx context.Context, id string) error {
	return s.cargoRepo.Delete(ctx, id)
}

func (s *OrganigramaService) GetAllOficinas(ctx context.Context) ([]models.Oficina, error) {
	return s.oficinaRepo.FindAll(ctx)
}

func (s *OrganigramaService) CreateOficina(ctx context.Context, oficina *models.Oficina) error {
	return s.oficinaRepo.Create(ctx, oficina)
}

func (s *OrganigramaService) DeleteOficina(ctx context.Context, id string) error {
	return s.oficinaRepo.Delete(ctx, id)
}
