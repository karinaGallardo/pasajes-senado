package services

import (
	"context"
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

func (s *CompensacionService) GetAll(ctx context.Context) ([]models.Compensacion, error) {
	return s.compRepo.WithContext(ctx).FindAll()
}

func (s *CompensacionService) Create(ctx context.Context, comp *models.Compensacion) error {
	return s.compRepo.WithContext(ctx).Create(comp)
}

func (s *CompensacionService) GetByID(ctx context.Context, id string) (*models.Compensacion, error) {
	return s.compRepo.WithContext(ctx).FindByID(id)
}

func (s *CompensacionService) Update(ctx context.Context, comp *models.Compensacion) error {
	return s.compRepo.WithContext(ctx).Update(comp)
}

func (s *CompensacionService) GetAllCategorias(ctx context.Context) ([]models.CategoriaCompensacion, error) {
	return s.catRepo.WithContext(ctx).FindAll()
}

func (s *CompensacionService) CreateCategoria(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return s.catRepo.WithContext(ctx).Create(cat)
}

func (s *CompensacionService) UpdateCategoria(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return s.catRepo.WithContext(ctx).Update(cat)
}

func (s *CompensacionService) GetCategoriaByDepartamentoAndTipo(ctx context.Context, dep, tipo string) (*models.CategoriaCompensacion, error) {
	return s.catRepo.WithContext(ctx).FindByDepartamentoAndTipo(dep, tipo)
}

func (s *CompensacionService) DeleteCategoria(ctx context.Context, id string) error {
	return s.catRepo.WithContext(ctx).Delete(id)
}
