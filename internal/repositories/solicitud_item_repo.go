package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type SolicitudItemRepository struct {
	db *gorm.DB
}

func NewSolicitudItemRepository() *SolicitudItemRepository {
	return &SolicitudItemRepository{db: configs.DB}
}

func (r *SolicitudItemRepository) WithTx(tx *gorm.DB) *SolicitudItemRepository {
	return &SolicitudItemRepository{db: tx}
}

func (r *SolicitudItemRepository) WithContext(ctx context.Context) *SolicitudItemRepository {
	return &SolicitudItemRepository{db: r.db.WithContext(ctx)}
}

func (r *SolicitudItemRepository) Update(item *models.SolicitudItem) error {
	return r.db.Save(item).Error
}

func (r *SolicitudItemRepository) FindByID(id string) (*models.SolicitudItem, error) {
	var item models.SolicitudItem
	err := r.db.Preload("Origen").Preload("Destino").First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SolicitudItemRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&models.SolicitudItem{}).Where("id = ?", id).Update("estado_codigo", status).Error
}
