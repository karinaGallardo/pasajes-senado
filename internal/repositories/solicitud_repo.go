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
	err := configs.DB.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("TipoSolicitud.ConceptoViaje").Order("created_at desc").Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByID(id string) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := configs.DB.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("Pasajes").Preload("TipoSolicitud.ConceptoViaje").First(&solicitud, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &solicitud, nil
}

func (r *SolicitudRepository) Update(solicitud *models.Solicitud) error {
	return configs.DB.Save(solicitud).Error
}

func (r *SolicitudRepository) ExistsByCodigo(codigo string) (bool, error) {
	var count int64
	err := configs.DB.Model(&models.Solicitud{}).Where("codigo = ?", codigo).Count(&count).Error
	return count > 0, err
}
