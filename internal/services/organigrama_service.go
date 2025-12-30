package services

import (
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

func (s *OrganigramaService) GetAllCargos() ([]models.Cargo, error) {
	return s.cargoRepo.FindAll()
}

func (s *OrganigramaService) CreateCargo(cargo *models.Cargo) error {
	return s.cargoRepo.Create(cargo)
}

func (s *OrganigramaService) DeleteCargo(id string) error {
	return s.cargoRepo.Delete(id)
}

func (s *OrganigramaService) GetAllOficinas() ([]models.Oficina, error) {
	return s.oficinaRepo.FindAll()
}

func (s *OrganigramaService) CreateOficina(oficina *models.Oficina) error {
	return s.oficinaRepo.Create(oficina)
}

func (s *OrganigramaService) DeleteOficina(id string) error {
	return s.oficinaRepo.Delete(id)
}
