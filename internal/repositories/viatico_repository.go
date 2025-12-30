package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type ViaticoRepository interface {
	Create(viatico *models.Viatico) error
	Update(viatico *models.Viatico) error
	Delete(id string) error
	FindByID(id string) (*models.Viatico, error)
	FindBySolicitudID(solicitudID string) ([]models.Viatico, error)
}

type viaticoRepository struct {
	db *gorm.DB
}

func NewViaticoRepository() ViaticoRepository {
	return &viaticoRepository{db: configs.DB}
}

func (r *viaticoRepository) Create(viatico *models.Viatico) error {
	return r.db.Create(viatico).Error
}

func (r *viaticoRepository) Update(viatico *models.Viatico) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(viatico).Error
}

func (r *viaticoRepository) Delete(id string) error {
	return r.db.Delete(&models.Viatico{}, "id = ?", id).Error
}

func (r *viaticoRepository) FindByID(id string) (*models.Viatico, error) {
	var viatico models.Viatico
	err := r.db.Preload("Detalles").Preload("Usuario").Preload("Solicitud").First(&viatico, "id = ?", id).Error
	return &viatico, err
}

func (r *viaticoRepository) FindBySolicitudID(solicitudID string) ([]models.Viatico, error) {
	var viaticos []models.Viatico
	err := r.db.Preload("Detalles").Where("solicitud_id = ?", solicitudID).Find(&viaticos).Error
	return viaticos, err
}
