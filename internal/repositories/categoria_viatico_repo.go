package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CategoriaViaticoRepository struct {
	db *gorm.DB
}

func NewCategoriaViaticoRepository(db *gorm.DB) *CategoriaViaticoRepository {
	return &CategoriaViaticoRepository{db: db}
}

func (r *CategoriaViaticoRepository) FindAll() ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.Order("codigo asc").Find(&list).Error
	return list, err
}

func (r *CategoriaViaticoRepository) FindByUbicacion(ubicacion string) ([]models.CategoriaViatico, error) {
	var list []models.CategoriaViatico
	err := r.db.Where("ubicacion = ?", ubicacion).Order("codigo asc").Find(&list).Error
	return list, err
}
