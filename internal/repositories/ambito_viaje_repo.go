package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type AmbitoViajeRepository struct {
	db *gorm.DB
}

func NewAmbitoViajeRepository(db *gorm.DB) *AmbitoViajeRepository {
	return &AmbitoViajeRepository{db: db}
}

func (r *AmbitoViajeRepository) FindAll() ([]models.AmbitoViaje, error) {
	var ambitos []models.AmbitoViaje
	err := r.db.Find(&ambitos).Error
	return ambitos, err
}
