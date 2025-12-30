package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type CompensacionService struct {
	compRepo *repositories.CompensacionRepository
	catRepo  *repositories.CategoriaCompensacionRepository
}

func NewCompensacionService() *CompensacionService {
	return &CompensacionService{
		compRepo: repositories.NewCompensacionRepository(),
		catRepo:  repositories.NewCategoriaCompensacionRepository(),
	}
}

func (s *CompensacionService) GetAll() ([]models.Compensacion, error) {
	return s.compRepo.FindAll()
}

func (s *CompensacionService) Create(comp *models.Compensacion) error {
	return s.compRepo.Create(comp)
}

func (s *CompensacionService) FindByID(id string) (*models.Compensacion, error) {
	return s.compRepo.FindByID(id)
}

func (s *CompensacionService) Update(comp *models.Compensacion) error {
	return s.compRepo.Update(comp)
}

func (s *CompensacionService) GetAllCategorias() ([]models.CategoriaCompensacion, error) {
	return s.catRepo.FindAll()
}

func (s *CompensacionService) SaveCategoria(cat *models.CategoriaCompensacion) error {
	return s.catRepo.Save(cat)
}

func (s *CompensacionService) FindCategoriaByDepartamentoAndTipo(dep, tipo string) (*models.CategoriaCompensacion, error) {
	return s.catRepo.FindByDepartamentoAndTipo(dep, tipo)
}

func (s *CompensacionService) DeleteCategoria(id string) error {
	return s.catRepo.Delete(id)
}
