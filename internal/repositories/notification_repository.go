package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) WithTx(tx *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: tx}
}

func (r *NotificationRepository) WithContext(ctx context.Context) *NotificationRepository {
	return &NotificationRepository{db: r.db.WithContext(ctx)}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NotificationRepository) FindByUserID(ctx context.Context, userID string, limit int) ([]models.Notification, error) {
	var notifications []models.Notification
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&models.Notification{}).Where("id = ?", id).Update("is_read", true).Error
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}
