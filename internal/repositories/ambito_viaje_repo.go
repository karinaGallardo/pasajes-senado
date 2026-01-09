package repositories

import (
	"context"
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

func (r *AmbitoViajeRepository) WithContext(ctx context.Context) *AmbitoViajeRepository {
	return &AmbitoViajeRepository{db: r.db.WithContext(ctx)}
}

func (r *AmbitoViajeRepository) FindAll() ([]models.AmbitoViaje, error) {
	var ambitos []models.AmbitoViaje
	err := r.db.Find(&ambitos).Error
	return ambitos, err
}
func (r *AmbitoViajeRepository) FindByCodigo(codigo string) (*models.AmbitoViaje, error) {
	var ambito models.AmbitoViaje
	err := r.db.Where("codigo = ?", codigo).First(&ambito).Error
	return &ambito, err
}
