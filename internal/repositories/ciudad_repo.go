package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CiudadRepository struct {
	db *gorm.DB
}

func NewCiudadRepository(db *gorm.DB) *CiudadRepository {
	return &CiudadRepository{db: db}
}

func (r *CiudadRepository) FindAll() ([]models.Ciudad, error) {
	var ciudades []models.Ciudad
	err := r.db.Order("nombre asc").Find(&ciudades).Error
	return ciudades, err
}

func (r *CiudadRepository) Create(destino *models.Ciudad) error {
	return r.db.Create(destino).Error
}
