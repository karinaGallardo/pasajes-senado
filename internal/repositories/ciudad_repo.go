package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type CiudadRepository struct{}

func NewCiudadRepository() *CiudadRepository {
	return &CiudadRepository{}
}

func (r *CiudadRepository) FindAll() ([]models.Ciudad, error) {
	var ciudades []models.Ciudad
	err := configs.DB.Order("nombre asc").Find(&ciudades).Error
	return ciudades, err
}

func (r *CiudadRepository) Create(destino *models.Ciudad) error {
	return configs.DB.Create(destino).Error
}
