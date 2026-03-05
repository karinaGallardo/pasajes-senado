package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type ZonaViaticoRepository struct {
	db *gorm.DB
}

func NewZonaViaticoRepository(db *gorm.DB) *ZonaViaticoRepository {
	return &ZonaViaticoRepository{db: db}
}

func (r *ZonaViaticoRepository) WithContext(ctx context.Context) *ZonaViaticoRepository {
	return &ZonaViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *ZonaViaticoRepository) FindAll(ctx context.Context) ([]models.ZonaViatico, error) {
	var list []models.ZonaViatico
	err := r.db.WithContext(ctx).Order("nombre asc").Find(&list).Error
	return list, err
}

func (r *ZonaViaticoRepository) Create(ctx context.Context, zona *models.ZonaViatico) error {
	return r.db.WithContext(ctx).Create(zona).Error
}
