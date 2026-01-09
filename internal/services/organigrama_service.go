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

func NewOrganigramaService() *OrganigramaService {
	return &OrganigramaService{
		cargoRepo:   repositories.NewCargoRepository(),
		oficinaRepo: repositories.NewOficinaRepository(),
	}
}

func (s *OrganigramaService) GetAllCargos(ctx context.Context) ([]models.Cargo, error) {
	return s.cargoRepo.WithContext(ctx).FindAll()
}

func (s *OrganigramaService) CreateCargo(ctx context.Context, cargo *models.Cargo) error {
	return s.cargoRepo.WithContext(ctx).Create(cargo)
}

func (s *OrganigramaService) DeleteCargo(ctx context.Context, id string) error {
	return s.cargoRepo.WithContext(ctx).Delete(id)
}

func (s *OrganigramaService) GetAllOficinas(ctx context.Context) ([]models.Oficina, error) {
	return s.oficinaRepo.WithContext(ctx).FindAll()
}

func (s *OrganigramaService) CreateOficina(ctx context.Context, oficina *models.Oficina) error {
	return s.oficinaRepo.WithContext(ctx).Create(oficina)
}

func (s *OrganigramaService) DeleteOficina(ctx context.Context, id string) error {
	return s.oficinaRepo.WithContext(ctx).Delete(id)
}
