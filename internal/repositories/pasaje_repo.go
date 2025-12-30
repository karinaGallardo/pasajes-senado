package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type PasajeRepository struct {
	db *gorm.DB
}

func NewPasajeRepository(db *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: db}
}

func (r *PasajeRepository) Create(pasaje *models.Pasaje) error {
	return r.db.Create(pasaje).Error
}

func (r *PasajeRepository) FindBySolicitudID(solicitudID string) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	err := r.db.Where("solicitud_id = ?", solicitudID).Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) Delete(id uint) error {
	return r.db.Delete(&models.Pasaje{}, id).Error
}
