package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type SolicitudRepository struct{}

func NewSolicitudRepository() *SolicitudRepository {
	return &SolicitudRepository{}
}

func (r *SolicitudRepository) Create(solicitud *models.Solicitud) error {
	return configs.DB.Create(solicitud).Error
}

func (r *SolicitudRepository) FindAll() ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := configs.DB.Preload("Usuario").Preload("Origen").Preload("Destino").Order("created_at desc").Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByID(id uint) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := configs.DB.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("Pasajes").First(&solicitud, id).Error
	if err != nil {
		return nil, err
	}
	return &solicitud, nil
}
