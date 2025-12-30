package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type CiudadRepository struct {
	db *gorm.DB
}

func NewCiudadRepository() *CiudadRepository {
	return &CiudadRepository{db: configs.DB}
}

func (r *CiudadRepository) FindAll() ([]models.Ciudad, error) {
	var ciudades []models.Ciudad
	err := r.db.Order("nombre asc").Find(&ciudades).Error
	return ciudades, err
}

func (r *CiudadRepository) Create(destino *models.Ciudad) error {
	return r.db.Create(destino).Error
}
