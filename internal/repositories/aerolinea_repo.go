package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type AerolineaRepository struct {
	db *gorm.DB
}

func NewAerolineaRepository(db *gorm.DB) *AerolineaRepository {
	return &AerolineaRepository{db: db}
}

func (r *AerolineaRepository) WithContext(ctx context.Context) *AerolineaRepository {
	return &AerolineaRepository{db: r.db.WithContext(ctx)}
}

func (r *AerolineaRepository) FindAllActive(ctx context.Context) ([]models.Aerolinea, error) {
	var aerolineas []models.Aerolinea
	err := r.db.WithContext(ctx).Where("estado = ?", true).Find(&aerolineas).Error
	return aerolineas, err
}

func (r *AerolineaRepository) FindAll(ctx context.Context) ([]models.Aerolinea, error) {
	var aerolineas []models.Aerolinea
	err := r.db.WithContext(ctx).Find(&aerolineas).Error
	return aerolineas, err
}

func (r *AerolineaRepository) Create(ctx context.Context, a *models.Aerolinea) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *AerolineaRepository) FindByID(ctx context.Context, id string) (*models.Aerolinea, error) {
	var a models.Aerolinea
	err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error
	return &a, err
}

func (r *AerolineaRepository) Update(ctx context.Context, a *models.Aerolinea) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *AerolineaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Aerolinea{}, "id = ?", id).Error
}
