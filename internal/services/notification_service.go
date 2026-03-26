package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type NotificationService struct {
	repo        *repositories.NotificationRepository
	usuarioRepo *repositories.UsuarioRepository
	pushService *PushService
}

func NewNotificationService(repo *repositories.NotificationRepository, usuarioRepo *repositories.UsuarioRepository, pushService *PushService) *NotificationService {
	return &NotificationService{
		repo:        repo,
		usuarioRepo: usuarioRepo,
		pushService: pushService,
	}
}

func (s *NotificationService) GetRecentByUserID(ctx context.Context, userID string) ([]models.Notification, error) {
	return s.repo.FindByUserID(ctx, userID, 10)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, id string) error {
	return s.repo.MarkAsRead(ctx, id)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllAsRead(ctx, userID)
}

func (s *NotificationService) NotifyAdmins(ctx context.Context, title, message, notifType, targetURL string) error {
	admins, err := s.usuarioRepo.FindAdminsAndResponsables(ctx)
	if err != nil {
		return err
	}

	for _, admin := range admins {
		notif := models.Notification{
			UserID:    admin.ID,
			Title:     title,
			Message:   message,
			Type:      notifType,
			TargetURL: targetURL,
		}
		if err := s.repo.Create(ctx, &notif); err == nil {
			// Broadcast via WebSocket
			Hub.Broadcast(map[string]interface{}{
				"event":       "refresh_notifications",
				"target_user": admin.ID,
				"title":       title,
				"message":     message,
				"type":        notifType,
				"url":         targetURL,
			})

			// Enviar Push a móvil (nuevo)
			s.pushService.SendToUser(ctx, admin.ID, title, message, targetURL)
		}
	}
	return nil
}
