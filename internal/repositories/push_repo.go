package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type PushRepository struct {
	db *gorm.DB
}

func NewPushRepository(db *gorm.DB) *PushRepository {
	return &PushRepository{db: db}
}

func (r *PushRepository) Create(ctx context.Context, sub *models.PushSubscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *PushRepository) Update(ctx context.Context, sub *models.PushSubscription) error {
	return r.db.WithContext(ctx).Save(sub).Error
}

func (r *PushRepository) FindByEndpoint(ctx context.Context, endpoint string) (*models.PushSubscription, error) {
	var sub models.PushSubscription
	err := r.db.WithContext(ctx).Where("endpoint = ?", endpoint).First(&sub).Error
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *PushRepository) FindByUserID(ctx context.Context, userID string) ([]models.PushSubscription, error) {
	var subs []models.PushSubscription
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&subs).Error
	return subs, err
}

func (r *PushRepository) Delete(ctx context.Context, endpoint string) error {
	return r.db.WithContext(ctx).Where("endpoint = ?", endpoint).Delete(&models.PushSubscription{}).Error
}
