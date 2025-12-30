package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DescargoRepository struct {
	db *gorm.DB
}

func NewDescargoRepository(db *gorm.DB) *DescargoRepository {
	return &DescargoRepository{db: db}
}

func (r *DescargoRepository) Create(descargo *models.Descargo) error {
	return r.db.Create(descargo).Error
}

func (r *DescargoRepository) FindBySolicitudID(solicitudID string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.Preload("Documentos").Where("solicitud_id = ?", solicitudID).First(&descargo).Error
	return &descargo, err
}

func (r *DescargoRepository) FindByID(id string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.Preload("Documentos").Preload("Solicitud").Preload("Solicitud.Usuario").Preload("Solicitud.Origen").Preload("Solicitud.Destino").First(&descargo, "id = ?", id).Error
	return &descargo, err
}

func (r *DescargoRepository) FindAll() ([]models.Descargo, error) {
	var descargos []models.Descargo
	err := r.db.Preload("Solicitud").Preload("Solicitud.Usuario").Order("created_at desc").Find(&descargos).Error
	return descargos, err
}
func (r *DescargoRepository) Update(descargo *models.Descargo) error {
	return r.db.Save(descargo).Error
}
