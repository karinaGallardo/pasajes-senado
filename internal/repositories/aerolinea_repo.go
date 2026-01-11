package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type AerolineaRepository struct {
	db *gorm.DB
}

func NewAerolineaRepository() *AerolineaRepository {
	return &AerolineaRepository{db: configs.DB}
}

func (r *AerolineaRepository) WithContext(ctx context.Context) *AerolineaRepository {
	return &AerolineaRepository{db: r.db.WithContext(ctx)}
}

func (r *AerolineaRepository) FindAllActive() ([]models.Aerolinea, error) {
	var aerolineas []models.Aerolinea
	err := r.db.Where("estado = ?", true).Find(&aerolineas).Error
	return aerolineas, err
}

func (r *AerolineaRepository) FindAll() ([]models.Aerolinea, error) {
	var aerolineas []models.Aerolinea
	err := r.db.Find(&aerolineas).Error
	return aerolineas, err
}

func (r *AerolineaRepository) Create(a *models.Aerolinea) error {
	return r.db.Create(a).Error
}

func (r *AerolineaRepository) FindByID(id string) (*models.Aerolinea, error) {
	var a models.Aerolinea
	err := r.db.First(&a, "id = ?", id).Error
	return &a, err
}

func (r *AerolineaRepository) Save(a *models.Aerolinea) error {
	return r.db.Save(a).Error
}

func (r *AerolineaRepository) Delete(id string) error {
	return r.db.Delete(&models.Aerolinea{}, "id = ?", id).Error
}
