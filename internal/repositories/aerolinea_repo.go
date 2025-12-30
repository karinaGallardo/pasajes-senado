package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type AerolineaRepository struct {
	db *gorm.DB
}

func NewAerolineaRepository(db *gorm.DB) *AerolineaRepository {
	return &AerolineaRepository{db: db}
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
