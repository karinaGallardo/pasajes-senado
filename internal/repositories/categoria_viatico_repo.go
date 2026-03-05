package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type CategoriaViaticoRepository struct {
	db *gorm.DB
}

func NewCategoriaViaticoRepository(db *gorm.DB) *CategoriaViaticoRepository {
	return &CategoriaViaticoRepository{db: db}
}

func (r *CategoriaViaticoRepository) WithContext(ctx context.Context) *CategoriaViaticoRepository {
	return &CategoriaViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *CategoriaViaticoRepository) FindAll(ctx context.Context) ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.WithContext(ctx).Preload("ZonaViatico").Order("codigo asc").Find(&list).Error
	return list, err
}

func (r *CategoriaViaticoRepository) FindByUbicacion(ctx context.Context, ubicacion string) ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.WithContext(ctx).Where("ubicacion = ?", ubicacion).Order("codigo asc").Find(&list).Error
	return list, err
}

func (r *CategoriaViaticoRepository) Create(ctx context.Context, cat *models.CategoriaViatico) error {
	return r.db.WithContext(ctx).Create(cat).Error
}
