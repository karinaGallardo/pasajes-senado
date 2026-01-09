package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DepartamentoRepository struct {
	db *gorm.DB
}

func NewDepartamentoRepository() *DepartamentoRepository {
	return &DepartamentoRepository{db: configs.DB}
}

func (r *DepartamentoRepository) WithContext(ctx context.Context) *DepartamentoRepository {
	return &DepartamentoRepository{db: r.db.WithContext(ctx)}
}

func (r *DepartamentoRepository) FindAll() ([]models.Departamento, error) {
	var depts []models.Departamento
	err := r.db.Order("nombre asc").Find(&depts).Error
	return depts, err
}

func (r *DepartamentoRepository) FindByNombre(nombre string) (*models.Departamento, error) {
	var dept models.Departamento
	err := r.db.Where("LOWER(nombre) = LOWER(?)", nombre).First(&dept).Error
	return &dept, err
}

func (r *DepartamentoRepository) FirstOrCreate(dept *models.Departamento) error {
	return r.db.Where("codigo = ?", dept.Codigo).FirstOrCreate(dept).Error
}
