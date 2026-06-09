package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
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

func (s *CompensacionService) CreateFromRequest(ctx context.Context, req dtos.CreateCompensacionRequest) (*models.Compensacion, error) {
	fechaInicio, _ := time.Parse("2006-01-02", req.FechaInicio)
	fechaFin, _ := time.Parse("2006-01-02", req.FechaFin)
	total, _ := strconv.ParseFloat(req.Total, 64)
	retencion, _ := strconv.ParseFloat(req.Retencion, 64)

	const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	codeSuffix, _ := gonanoid.Generate(alphabet, 6)
	codigo := fmt.Sprintf("COMP-%d-%s", time.Now().Year(), codeSuffix)

	comp := &models.Compensacion{
		Codigo:          codigo,
		Nombre:          req.NombreTramite,
		FuncionarioID:   req.FuncionarioID,
		FechaInicio:     fechaInicio,
		FechaFin:        fechaFin,
		MesCompensacion: req.Mes,
		Estado:          "BORRADOR",
		Glosa:           req.Glosa,
		Total:           total,
		Retencion:       retencion,
		Informe:         req.Informe,
	}

	if err := s.compRepo.Create(ctx, comp); err != nil {
		return nil, err
	}
	return comp, nil
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
