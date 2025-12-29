package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type PasajeRepository struct{}

func NewPasajeRepository() *PasajeRepository {
	return &PasajeRepository{}
}

func (r *PasajeRepository) Create(pasaje *models.Pasaje) error {
	return configs.DB.Create(pasaje).Error
}

func (r *PasajeRepository) FindBySolicitudID(solicitudID string) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	err := configs.DB.Where("solicitud_id = ?", solicitudID).Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) Delete(id uint) error {
	return configs.DB.Delete(&models.Pasaje{}, id).Error
}
