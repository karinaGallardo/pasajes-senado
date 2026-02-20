package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type ViaticoRepository struct {
	db *gorm.DB
}

func NewViaticoRepository() *ViaticoRepository {
	return &ViaticoRepository{db: configs.DB}
}

func (r *ViaticoRepository) WithTx(tx *gorm.DB) *ViaticoRepository {
	return &ViaticoRepository{db: tx}
}

func (r *ViaticoRepository) WithContext(ctx context.Context) *ViaticoRepository {
	return &ViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *ViaticoRepository) Create(viatico *models.Viatico) error {
	return r.db.Create(viatico).Error
}

func (r *ViaticoRepository) Update(viatico *models.Viatico) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(viatico).Error
}

func (r *ViaticoRepository) Delete(id string) error {
	return r.db.Delete(&models.Viatico{}, "id = ?", id).Error
}

func (r *ViaticoRepository) FindByID(id string) (*models.Viatico, error) {
	var viatico models.Viatico
	err := r.db.
		Preload("Detalles").
		Preload("Usuario").
		Preload("Usuario.Rol").
		Preload("Solicitud").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		First(&viatico, "id = ?", id).Error
	return &viatico, err
}

func (r *ViaticoRepository) FindBySolicitudID(solicitudID string) ([]models.Viatico, error) {
	var viaticos []models.Viatico
	err := r.db.Preload("Detalles").Where("solicitud_id = ?", solicitudID).Find(&viaticos).Error
	return viaticos, err
}

func (r *ViaticoRepository) FindAll() ([]models.Viatico, error) {
	var list []models.Viatico
	err := r.db.Preload("Usuario").Preload("Solicitud").Order("fecha_asignacion desc").Find(&list).Error
	return list, err
}
