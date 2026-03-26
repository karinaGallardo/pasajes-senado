package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *AuditRepository) FindByEntity(ctx context.Context, entityType, entityID string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := r.db.WithContext(ctx).
		Preload("Usuario").
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

func (r *AuditRepository) FindAll(ctx context.Context, filters map[string]string, limit, offset int) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.AuditLog{}).Preload("Usuario")

	if val, ok := filters["action"]; ok && val != "" {
		query = query.Where("action = ?", val)
	}
	if val, ok := filters["entity_type"]; ok && val != "" {
		query = query.Where("entity_type = ?", val)
	}
	if val, ok := filters["user_id"]; ok && val != "" {
		query = query.Where("user_id = ?", val)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}
