package services

import (
	"context"
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type OpenTicketService struct {
	repo          *repositories.OpenTicketRepository
	solicitudRepo *repositories.SolicitudRepository
	usuarioRepo   *repositories.UsuarioRepository
	pasajeRepo    *repositories.PasajeRepository
}

func NewOpenTicketService(
	repo *repositories.OpenTicketRepository,
	solicitudRepo *repositories.SolicitudRepository,
	usuarioRepo *repositories.UsuarioRepository,
	pasajeRepo *repositories.PasajeRepository,
) *OpenTicketService {
	return &OpenTicketService{
		repo:          repo,
		solicitudRepo: solicitudRepo,
		usuarioRepo:   usuarioRepo,
		pasajeRepo:    pasajeRepo,
	}
}

func (s *OpenTicketService) GetByID(ctx context.Context, id string) (*models.OpenTicket, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *OpenTicketService) GetDisponiblesByUsuarioID(ctx context.Context, usuarioID string) ([]models.OpenTicket, error) {
	return s.repo.FindDisponiblesByUsuarioID(ctx, usuarioID)
}

func (s *OpenTicketService) GetAll(ctx context.Context, filters map[string]any) ([]models.OpenTicket, error) {
	return s.repo.FindAll(ctx, filters)
}

func (s *OpenTicketService) GetPendientesByUserIDs(ctx context.Context, userIDs []string) (int64, error) {
	return s.repo.CountByEstado(ctx, models.EstadoOpenTicketPendiente, userIDs)
}

// GetScoped obtiene los tickets según el rol del usuario:
// - Admin/Responsable: todos los tickets
// - Senador: solo los propios
// - Funcionario: los de sus beneficiarios asignados
func (s *OpenTicketService) GetScoped(ctx context.Context, user *models.Usuario) ([]models.OpenTicket, error) {
	if user.IsAdminOrResponsable() {
		return s.repo.FindAll(ctx, nil)
	}
	if user.IsSenador() {
		return s.repo.FindAllByUsuarioID(ctx, user.ID)
	}
	managedUsers, _ := s.usuarioRepo.FindByEncargadoID(ctx, user.ID)
	if len(managedUsers) == 0 {
		return nil, nil
	}
	managedIDs := make([]string, 0, len(managedUsers))
	for _, u := range managedUsers {
		managedIDs = append(managedIDs, u.ID)
	}
	var allTickets []models.OpenTicket
	for _, id := range managedIDs {
		tks, _ := s.repo.FindAllByUsuarioID(ctx, id)
		allTickets = append(allTickets, tks...)
	}
	return allTickets, nil
}

// SyncFromDescargo sincroniza los tramos marcados como OpenTicket en el descargo
// con la tabla de open_tickets. Se ejecuta cuando un descargo se envía a revisión.
func (s *OpenTicketService) SyncFromDescargo(ctx context.Context, descargo *models.Descargo, userID string) error {
	if descargo.Solicitud != nil && descargo.Solicitud.GetConceptoCodigo() == "OFICIAL" {
		return nil
	}

	existing, _ := s.repo.FindByDescargoID(ctx, descargo.ID)
	existingMap := make(map[string]models.OpenTicket)
	for _, t := range existing {
		existingMap[t.NumeroBillete] = t
	}

	openTicketsInDescargo := make(map[string]models.OpenTicket)
	for _, tramo := range descargo.Tramos {
		if !tramo.EsOpenTicket || tramo.Billete == "" {
			continue
		}
		ticket, ok := openTicketsInDescargo[tramo.Billete]
		if !ok {
			ticket = models.OpenTicket{
				UsuarioID:      descargo.UsuarioID,
				DescargoID:     descargo.ID,
				NumeroBillete:  tramo.Billete,
				Estado:         models.EstadoOpenTicketPendiente,
				Observaciones:  "Generado automáticamente desde descargo.",
				TramosNoUsados: "",
			}
			if tramo.PasajeID != nil && *tramo.PasajeID != "" {
				ticket.PasajeID = tramo.PasajeID
			} else if tramo.Billete != "" {
				if p, err := s.pasajeRepo.FindByNumeroBillete(ctx, tramo.Billete); err == nil {
					ticket.PasajeID = &p.ID
				}
			}
		}

		if ticket.TramosNoUsados != "" {
			ticket.TramosNoUsados += " / "
		}
		ticket.TramosNoUsados += tramo.TramoNombre
		openTicketsInDescargo[tramo.Billete] = ticket
	}

	for num, ticket := range openTicketsInDescargo {
		if old, ok := existingMap[num]; ok {
			if old.Estado == models.EstadoOpenTicketPendiente || old.Estado == models.EstadoOpenTicketDisponible {
				old.TramosNoUsados = ticket.TramosNoUsados
				old.PasajeID = ticket.PasajeID
				old.UpdatedBy = &userID
				_ = s.repo.Update(ctx, &old)
			}
			delete(existingMap, num)
		} else {
			ticket.CreatedBy = &userID
			_ = s.repo.Create(ctx, &ticket)
		}
	}

	for _, old := range existingMap {
		if old.Estado == models.EstadoOpenTicketPendiente {
			_ = s.repo.Delete(ctx, old.ID)
		}
	}

	return nil
}

// Approve cambia un ticket de PENDIENTE a DISPONIBLE (hecho por la encargada de pasajes).
func (s *OpenTicketService) Approve(ctx context.Context, id string, user *models.Usuario) error {
	ticket, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if ticket.Estado != models.EstadoOpenTicketPendiente {
		return errors.New("solo se pueden aprobar tickets en estado PENDIENTE")
	}
	ticket.Estado = models.EstadoOpenTicketDisponible
	ticket.UpdatedBy = &user.ID
	return s.repo.Update(ctx, ticket)
}

// ReserveAsigna un ticket DISPONIBLE a una solicitud (RESERVADO).
func (s *OpenTicketService) Reserve(ctx context.Context, id, solicitudID string, user *models.Usuario) error {
	ticket, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if ticket.Estado != models.EstadoOpenTicketDisponible {
		return errors.New("solo se pueden reservar tickets DISPONIBLES")
	}
	ticket.Estado = models.EstadoOpenTicketReservado
	ticket.SolicitudConsumoID = &solicitudID
	ticket.UpdatedBy = &user.ID
	return s.repo.Update(ctx, ticket)
}

// Finalize marca un ticket como FINALIZADO (ya usado en un pasaje).
func (s *OpenTicketService) Finalize(ctx context.Context, id string) error {
	ticket, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	ticket.Estado = models.EstadoOpenTicketFinalizado
	return s.repo.Update(ctx, ticket)
}

// RevertToDisponible revierte un ticket RESERVADO/FINALIZADO a DISPONIBLE.
func (s *OpenTicketService) RevertToDisponible(ctx context.Context, id string) error {
	ticket, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if ticket.Estado != models.EstadoOpenTicketReservado && ticket.Estado != models.EstadoOpenTicketFinalizado {
		return errors.New("solo se pueden revertir tickets RESERVADOS o FINALIZADOS")
	}
	ticket.Estado = models.EstadoOpenTicketDisponible
	ticket.SolicitudConsumoID = nil
	return s.repo.Update(ctx, ticket)
}

// DeletePending elimina tickets PENDIENTES (al borrar descargo).
func (s *OpenTicketService) DeletePending(ctx context.Context, descargoID string) error {
	tickets, err := s.repo.FindByDescargoID(ctx, descargoID)
	if err != nil {
		return err
	}
	for _, t := range tickets {
		if t.Estado == models.EstadoOpenTicketPendiente {
			_ = s.repo.Delete(ctx, t.ID)
		}
	}
	return nil
}

// DescargoOTs busca los Open Tickets asociados a un descargo.
func (s *OpenTicketService) DescargoOTs(ctx context.Context, descargoID string) ([]models.OpenTicket, error) {
	return s.repo.FindByDescargoID(ctx, descargoID)
}

// Update persiste cambios en un ticket existente.
func (s *OpenTicketService) Update(ctx context.Context, ticket *models.OpenTicket) error {
	return s.repo.Update(ctx, ticket)
}
