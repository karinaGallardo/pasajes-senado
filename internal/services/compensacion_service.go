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

func NewCompensacionService(
	compRepo *repositories.CompensacionRepository,
	catRepo *repositories.CategoriaCompensacionRepository,
) *CompensacionService {
	return &CompensacionService{
		compRepo: compRepo,
		catRepo:  catRepo,
	}
}

func (s *CompensacionService) GetAll(ctx context.Context) ([]models.Compensacion, error) {
	return s.compRepo.FindAll(ctx)
}

func (s *CompensacionService) Create(ctx context.Context, comp *models.Compensacion) error {
	return s.compRepo.Create(ctx, comp)
}

func (s *CompensacionService) GetByID(ctx context.Context, id string) (*models.Compensacion, error) {
	return s.compRepo.FindByID(ctx, id)
}

func (s *CompensacionService) Update(ctx context.Context, comp *models.Compensacion) error {
	return s.compRepo.Update(ctx, comp)
}

func (s *CompensacionService) GetAllCategorias(ctx context.Context) ([]models.CategoriaCompensacion, error) {
	return s.catRepo.FindAll(ctx)
}

func (s *CompensacionService) CreateCategoria(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return s.catRepo.Create(ctx, cat)
}

func (s *CompensacionService) UpdateCategoria(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return s.catRepo.Update(ctx, cat)
}

func (s *CompensacionService) GetCategoriaByDepartamentoAndTipo(ctx context.Context, dep, tipo string) (*models.CategoriaCompensacion, error) {
	return s.catRepo.FindByDepartamentoAndTipo(ctx, dep, tipo)
}

func (s *CompensacionService) DeleteCategoria(ctx context.Context, id string) error {
	return s.catRepo.Delete(ctx, id)
}
