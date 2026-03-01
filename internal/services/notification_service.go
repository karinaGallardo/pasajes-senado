package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type NotificationService struct {
	repo        *repositories.NotificationRepository
	usuarioRepo *repositories.UsuarioRepository
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		repo:        repositories.NewNotificationRepository(),
		usuarioRepo: repositories.NewUsuarioRepository(),
	}
}

func (s *NotificationService) GetRecentByUserID(ctx context.Context, userID string) ([]models.Notification, error) {
	return s.repo.FindByUserID(userID, 10)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnread(userID)
}

func (s *NotificationService) MarkAsRead(ctx context.Context, id string) error {
	return s.repo.MarkAsRead(id)
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllAsRead(userID)
}

func (s *NotificationService) NotifyAdmins(ctx context.Context, title, message, notifType, targetURL string) error {
	admins, err := s.usuarioRepo.FindAdminsAndResponsables()
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
		if err := s.repo.Create(&notif); err == nil {
			// Broadcast via WebSocket
			Hub.Broadcast(map[string]interface{}{
				"event":       "refresh_notifications",
				"target_user": admin.ID,
				"title":       title,
				"message":     message,
				"type":        notifType,
				"url":         targetURL,
			})
		}
	}
	return nil
}
