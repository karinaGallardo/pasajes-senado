package services

import (
	"context"
	"encoding/json"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/spf13/viper"
)

type PushService struct {
	repo *repositories.PushRepository
}

func NewPushService(repo *repositories.PushRepository) *PushService {
	return &PushService{repo: repo}
}

type PushSubscriptionDTO struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (s *PushService) Subscribe(ctx context.Context, userID string, dto PushSubscriptionDTO) error {
	existing, err := s.repo.FindByEndpoint(ctx, dto.Endpoint)
	if err == nil && existing != nil {
		existing.UserID = userID
		existing.P256dh = dto.Keys.P256dh
		existing.Auth = dto.Keys.Auth
		return s.repo.Update(ctx, existing)
	}

	sub := &models.PushSubscription{
		UserID:   userID,
		Endpoint: dto.Endpoint,
		P256dh:   dto.Keys.P256dh,
		Auth:     dto.Keys.Auth,
	}
	return s.repo.Create(ctx, sub)
}

func (s *PushService) SendToUser(ctx context.Context, userID string, title, message, url string) error {
	subs, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	publicKey := viper.GetString("VAPID_PUBLIC_KEY")
	privateKey := viper.GetString("VAPID_PRIVATE_KEY")

	if publicKey == "" || privateKey == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]string{
		"title":   title,
		"message": message,
		"url":     url,
	})

	for _, sub := range subs {
		s.sendPush(sub, string(payload), publicKey, privateKey)
	}

	return nil
}

func (s *PushService) sendPush(sub models.PushSubscription, payload, pub, priv string) {
	sSub := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.P256dh,
			Auth:   sub.Auth,
		},
	}

	// Podríamos ejecutar en goroutine
	go func() {
		resp, err := webpush.SendNotification([]byte(payload), sSub, &webpush.Options{
			Subscriber:      "mailto:pasajes.go@senado.gob.bo",
			VAPIDPublicKey:  pub,
			VAPIDPrivateKey: priv,
			TTL:             30,
		})
		if err != nil {
			// Log error but continue
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 410 || resp.StatusCode == 404 {
				// Suscripción expirada, borrarla
				s.repo.Delete(context.Background(), sub.Endpoint)
			}
		}
	}()
}
