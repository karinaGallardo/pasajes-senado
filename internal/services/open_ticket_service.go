package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type OpenTicketService struct {
	repo *repositories.OpenTicketRepository
}

func NewOpenTicketService(repo *repositories.OpenTicketRepository) *OpenTicketService {
	return &OpenTicketService{repo: repo}
}

func (s *OpenTicketService) GetDisponiblesByUsuarioID(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	return s.repo.FindDisponiblesByUsuarioID(ctx, usuarioID)
}

func (s *OpenTicketService) GetByUsuarioID(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	return s.repo.FindAllByUsuarioID(ctx, usuarioID)
}

func (s *OpenTicketService) GetByID(ctx context.Context, id string) (*models.OpenTicket, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *OpenTicketService) GetAll(ctx context.Context, filters map[string]any) ([]models.OpenTicket, error) {
	return s.repo.FindAll(ctx, filters)
}

func (s *OpenTicketService) GetForUser(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	return s.repo.FindAllByUsuarioID(ctx, usuarioID)
}

func (s *OpenTicketService) Update(ctx context.Context, ticket *models.OpenTicket) error {
	return s.repo.Update(ctx, ticket)
}

func (s *OpenTicketService) GetPendingCount(ctx context.Context, userIDs []string) int64 {
	count, _ := s.repo.CountByEstado(ctx, models.EstadoOpenTicketPendiente, userIDs)
	return count
}
