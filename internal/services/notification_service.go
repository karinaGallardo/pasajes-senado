package services

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strings"
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

func (s *NotificationService) NotifySolicitudCreated(ctx context.Context, sol *models.Solicitud) error {
	title := "Nueva Solicitud: " + sol.Codigo

	benefName := sol.UsuarioID
	if sol.Usuario.ID != "" {
		benefName = sol.Usuario.GetNombreResumido()
	}

	concepto := "PASAJES"
	if sol.TipoSolicitudCodigo != "" {
		concepto = strings.ToUpper(sol.TipoSolicitudCodigo)
	}

	message := fmt.Sprintf("<ul class='list-none space-y-0.5 mt-1'><li><strong>Beneficiario:</strong> %s</li><li><strong>Fecha:</strong> %s</li><li><strong>Tipo:</strong> %s</li></ul>",
		benefName,
		sol.CreatedAt.Format("02/01/2006 15:04"),
		concepto)

	solPath := "derecho"
	if sol.IsOficial() {
		solPath = "oficial"
	}
	targetURL := fmt.Sprintf("/solicitudes/%s/%s/detalle", solPath, sol.ID)

	return s.NotifyAdmins(ctx, title, message, "new_solicitud", targetURL)
}
