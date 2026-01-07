package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type EstadoPasajeRepository struct {
	db *gorm.DB
}

func NewEstadoPasajeRepository() *EstadoPasajeRepository {
	return &EstadoPasajeRepository{db: configs.DB}
}

func (r *EstadoPasajeRepository) FindByCodigo(codigo string) (*models.EstadoPasaje, error) {
	var estado models.EstadoPasaje
	err := r.db.First(&estado, "codigo = ?", codigo).Error
	return &estado, err
}

func (r *EstadoPasajeRepository) FindAll() ([]models.EstadoPasaje, error) {
	var estados []models.EstadoPasaje
	err := r.db.Find(&estados).Error
	return estados, err
}
