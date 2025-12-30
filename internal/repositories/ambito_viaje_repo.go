package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type AmbitoViajeRepository struct {
	db *gorm.DB
}

func NewAmbitoViajeRepository() *AmbitoViajeRepository {
	return &AmbitoViajeRepository{db: configs.DB}
}

func (r *AmbitoViajeRepository) FindAll() ([]models.AmbitoViaje, error) {
	var ambitos []models.AmbitoViaje
	err := r.db.Find(&ambitos).Error
	return ambitos, err
}
