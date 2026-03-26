package services

import (
	"context"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type AuditService struct {
	repo *repositories.AuditRepository
}

func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(ctx context.Context, action, entityType, entityID, oldVal, newVal, ip, userAgent string) error {
	userID := appcontext.GetUserIDFromContext(ctx)
	
	entry := &models.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValue:   oldVal,
		NewValue:   newVal,
		UserID:     userID,
		IP:         ip,
		UserAgent:  userAgent,
	}

	return s.repo.Create(ctx, entry)
}

func (s *AuditService) GetHistory(ctx context.Context, entityType, entityID string) ([]models.AuditLog, error) {
	return s.repo.FindByEntity(ctx, entityType, entityID)
}
