package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type AmbitoViajeRepository struct {
	db *gorm.DB
}

func NewAmbitoViajeRepository(db *gorm.DB) *AmbitoViajeRepository {
	return &AmbitoViajeRepository{db: db}
}

func (r *AmbitoViajeRepository) WithContext(ctx context.Context) *AmbitoViajeRepository {
	return &AmbitoViajeRepository{db: r.db.WithContext(ctx)}
}

func (r *AmbitoViajeRepository) FindAll(ctx context.Context) ([]models.AmbitoViaje, error) {
	var ambitos []models.AmbitoViaje
	err := r.db.WithContext(ctx).Find(&ambitos).Error
	return ambitos, err
}
func (r *AmbitoViajeRepository) FindByCodigo(ctx context.Context, codigo string) (*models.AmbitoViaje, error) {
	var ambito models.AmbitoViaje
	err := r.db.WithContext(ctx).Where("codigo = ?", codigo).First(&ambito).Error
	return &ambito, err
}
