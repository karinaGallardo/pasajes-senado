package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type DescargoRepository struct{}

func NewDescargoRepository() *DescargoRepository {
	return &DescargoRepository{}
}

func (r *DescargoRepository) Create(descargo *models.Descargo) error {
	return configs.DB.Create(descargo).Error
}

func (r *DescargoRepository) FindBySolicitudID(solicitudID string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := configs.DB.Where("solicitud_id = ?", solicitudID).First(&descargo).Error
	return &descargo, err
}

func (r *DescargoRepository) FindByID(id string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := configs.DB.Preload("Solicitud").Preload("Solicitud.Usuario").Preload("Solicitud.Origen").Preload("Solicitud.Destino").First(&descargo, id).Error
	return &descargo, err
}

func (r *DescargoRepository) FindAll() ([]models.Descargo, error) {
	var descargos []models.Descargo
	err := configs.DB.Preload("Solicitud").Preload("Solicitud.Usuario").Order("created_at desc").Find(&descargos).Error
	return descargos, err
}
