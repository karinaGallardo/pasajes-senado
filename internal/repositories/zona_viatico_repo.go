package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type ZonaViaticoRepository struct {
	db *gorm.DB
}

func NewZonaViaticoRepository() *ZonaViaticoRepository {
	return &ZonaViaticoRepository{db: configs.DB}
}

func (r *ZonaViaticoRepository) WithContext(ctx context.Context) *ZonaViaticoRepository {
	return &ZonaViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *ZonaViaticoRepository) FindAll() ([]models.ZonaViatico, error) {
	var list []models.ZonaViatico
	err := r.db.Order("nombre asc").Find(&list).Error
	return list, err
}

func (r *ZonaViaticoRepository) Create(zona *models.ZonaViatico) error {
	return r.db.Create(zona).Error
}
