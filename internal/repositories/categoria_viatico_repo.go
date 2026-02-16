package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type CategoriaViaticoRepository struct {
	db *gorm.DB
}

func NewCategoriaViaticoRepository() *CategoriaViaticoRepository {
	return &CategoriaViaticoRepository{db: configs.DB}
}

func (r *CategoriaViaticoRepository) WithContext(ctx context.Context) *CategoriaViaticoRepository {
	return &CategoriaViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *CategoriaViaticoRepository) FindAll() ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.Preload("ZonaViatico").Order("codigo asc").Find(&list).Error
	return list, err
}

func (r *CategoriaViaticoRepository) FindByUbicacion(ubicacion string) ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.Where("ubicacion = ?", ubicacion).Order("codigo asc").Find(&list).Error
	return list, err
}

func (r *CategoriaViaticoRepository) Create(cat *models.CategoriaViatico) error {
	return r.db.Create(cat).Error
}
