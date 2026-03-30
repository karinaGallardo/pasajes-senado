package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CategoriaCompensacionRepository struct {
	db *gorm.DB
}

func NewCategoriaCompensacionRepository(db *gorm.DB) *CategoriaCompensacionRepository {
	return &CategoriaCompensacionRepository{db: db}
}

func (r *CategoriaCompensacionRepository) WithContext(ctx context.Context) *CategoriaCompensacionRepository {
	return &CategoriaCompensacionRepository{db: r.db.WithContext(ctx)}
}

func (r *CategoriaCompensacionRepository) FindAll(ctx context.Context) ([]models.CategoriaCompensacion, error) {
	var list []models.CategoriaCompensacion
	err := r.db.WithContext(ctx).Order("departamento asc, tipo_senador asc").Find(&list).Error
	return list, err
}

func (r *CategoriaCompensacionRepository) Create(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return r.db.WithContext(ctx).Create(cat).Error
}

func (r *CategoriaCompensacionRepository) Update(ctx context.Context, cat *models.CategoriaCompensacion) error {
	return r.db.WithContext(ctx).Save(cat).Error
}

func (r *CategoriaCompensacionRepository) FindByDepartamentoAndTipo(ctx context.Context, dep, tipo string) (*models.CategoriaCompensacion, error) {
	var cat models.CategoriaCompensacion
	err := r.db.WithContext(ctx).Where("departamento = ? AND tipo_senador = ?", dep, tipo).First(&cat).Error
	return &cat, err
}

func (r *CategoriaCompensacionRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.CategoriaCompensacion{}, "id = ?", id).Error
}
