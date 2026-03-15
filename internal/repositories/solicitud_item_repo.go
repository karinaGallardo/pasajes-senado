package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type SolicitudItemRepository struct {
	db *gorm.DB
}

func NewSolicitudItemRepository(db *gorm.DB) *SolicitudItemRepository {
	return &SolicitudItemRepository{db: db}
}

func (r *SolicitudItemRepository) WithTx(tx *gorm.DB) *SolicitudItemRepository {
	return &SolicitudItemRepository{db: tx}
}

func (r *SolicitudItemRepository) WithContext(ctx context.Context) *SolicitudItemRepository {
	return &SolicitudItemRepository{db: r.db.WithContext(ctx)}
}

func (r *SolicitudItemRepository) Update(ctx context.Context, item *models.SolicitudItem) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *SolicitudItemRepository) FindByID(ctx context.Context, id string) (*models.SolicitudItem, error) {
	var item models.SolicitudItem
	err := r.db.WithContext(ctx).Preload("Solicitud").Preload("Origen").Preload("Destino").First(&item, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SolicitudItemRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.SolicitudItem{}).Where("id = ?", id).Update("estado_codigo", status).Error
}

func (r *SolicitudItemRepository) UpdateTimestamps(ctx context.Context, id string, createdAt, updatedAt any) error {
	return r.db.WithContext(ctx).Model(&models.SolicitudItem{}).Where("id = ?", id).Updates(map[string]interface{}{
		"created_at": createdAt,
		"updated_at": updatedAt,
	}).Error
}
