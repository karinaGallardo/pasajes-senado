package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type RutaRepository struct {
	db *gorm.DB
}

func NewRutaRepository(db *gorm.DB) *RutaRepository {
	return &RutaRepository{db: db}
}

func (r *RutaRepository) FindAll() ([]models.Ruta, error) {
	var rutas []models.Ruta
	err := r.db.Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Create(ruta *models.Ruta) error {
	return r.db.Create(ruta).Error
}

func (r *RutaRepository) FindByID(id string) (*models.Ruta, error) {
	var ruta models.Ruta
	err := r.db.First(&ruta, "id = ?", id).Error
	return &ruta, err
}
