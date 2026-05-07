package services

import (
	"context"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type AuditService struct {
	repo *repositories.AuditRepository
}

func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Log(ctx context.Context, action, entityType, entityID, oldVal, newVal, ip, userAgent string) error {
	userID := appcontext.GetUserIDFromContext(ctx)

	// Fallback a contexto si no se proveen explícitamente
	if ip == "" {
		ip = appcontext.GetIPFromContext(ctx)
	}
	if userAgent == "" {
		userAgent = appcontext.GetUserAgentFromContext(ctx)
	}

	safeAction := utils.TruncateString(action, 45)
	safeEntityType := utils.TruncateString(entityType, 45)
	safeIP := utils.TruncateString(ip, 45)
	var safeUserAgent string
	if userAgent != "" {
		safeUserAgent = utils.TruncateString(userAgent, 45)
		if len(userAgent) > 45 {
			safeUserAgent = utils.TruncateString(userAgent, 42) + "..."
		}
	}

	entry := &models.AuditLog{
		Action:     safeAction,
		EntityType: safeEntityType,
		EntityID:   entityID,
		OldValue:   oldVal,
		NewValue:   newVal,
		UserID:     userID,
		IP:         safeIP,
		UserAgent:  safeUserAgent,
	}

	return s.repo.Create(ctx, entry)
}

func (s *AuditService) GetHistory(ctx context.Context, entityType, entityID string) ([]models.AuditLog, error) {
	return s.repo.FindByEntity(ctx, entityType, entityID)
}

func (s *AuditService) GetAll(ctx context.Context, filters map[string]string, limit, offset int) ([]models.AuditLog, int64, error) {
	return s.repo.FindAll(ctx, filters, limit, offset)
}

func (s *AuditService) GetAvailableFilters(ctx context.Context) (actions []string, entities []string, err error) {
	actions = []string{"LOGIN", "LOGOUT", "CREAR_SOLICITUD", "ACTUALIZAR_SOLICITUD", "APROBAR_SOLICITUD", "RECHAZAR_SOLICITUD", "ACTUALIZAR_DESCARGO", "SUBMIT_DESCARGO", "APROBAR_DESCARGO"}
	entities = []string{"solicitud", "pasaje", "descargo", "usuario", "auth"}
	return
}
