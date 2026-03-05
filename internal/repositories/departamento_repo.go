package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DepartamentoRepository struct {
	db *gorm.DB
}

func NewDepartamentoRepository(db *gorm.DB) *DepartamentoRepository {
	return &DepartamentoRepository{db: db}
}

func (r *DepartamentoRepository) WithContext(ctx context.Context) *DepartamentoRepository {
	return &DepartamentoRepository{db: r.db.WithContext(ctx)}
}

func (r *DepartamentoRepository) FindAll(ctx context.Context) ([]models.Departamento, error) {
	var depts []models.Departamento
	err := r.db.WithContext(ctx).Order("nombre asc").Find(&depts).Error
	return depts, err
}

func (r *DepartamentoRepository) FindByNombre(ctx context.Context, nombre string) (*models.Departamento, error) {
	var dept models.Departamento
	err := r.db.WithContext(ctx).Where("LOWER(nombre) = LOWER(?)", nombre).First(&dept).Error
	return &dept, err
}

func (r *DepartamentoRepository) FirstOrCreate(ctx context.Context, dept *models.Departamento) error {
	return r.db.WithContext(ctx).Where("codigo = ?", dept.Codigo).FirstOrCreate(dept).Error
}
