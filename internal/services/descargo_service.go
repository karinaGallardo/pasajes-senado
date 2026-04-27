package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type DescargoService struct {
	repo             *repositories.DescargoRepository
	pasajeRepo       *repositories.PasajeRepository
	openTicketRepo   *repositories.OpenTicketRepository
	solicitudService *SolicitudService
	usuarioService   *UsuarioService
	auditService     *AuditService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	pasajeRepo *repositories.PasajeRepository,
	openTicketRepo *repositories.OpenTicketRepository,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
	auditService *AuditService,
) *DescargoService {
	return &DescargoService{
		repo:             repo,
		pasajeRepo:       pasajeRepo,
		openTicketRepo:   openTicketRepo,
		solicitudService: solicitudService,
		usuarioService:   usuarioService,
		auditService:     auditService,
	}
}

func (s *DescargoService) GetBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	return s.repo.FindBySolicitudID(ctx, solicitudID)
}

func (s *DescargoService) GetByID(ctx context.Context, id string) (*models.Descargo, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *DescargoService) GetAll(ctx context.Context) ([]models.Descargo, error) {
	return s.repo.FindAll(ctx)
}

func (s *DescargoService) GetCountByUserIDs(ctx context.Context, userIDs []string) int64 {
	count, _ := s.repo.FindCountByUserIDs(ctx, userIDs)
	return count
}

func (s *DescargoService) GetPaginated(ctx context.Context, page, limit int, searchTerm string, userIDs []string) (*repositories.PaginatedDescargos, error) {
	return s.repo.FindPaginated(ctx, page, limit, searchTerm, userIDs)
}

func (s *DescargoService) GetPaginatedScoped(ctx context.Context, authUser *models.Usuario, page, limit int, searchTerm string) (*repositories.PaginatedDescargos, error) {
	if authUser.IsAdminOrResponsable() {
		return s.repo.FindPaginated(ctx, page, limit, searchTerm, nil)
	}

	ids := []string{authUser.ID}
	if senators, err := s.usuarioService.GetSenatorsByEncargado(ctx, authUser.ID); err == nil {
		for _, sen := range senators {
			ids = append(ids, sen.ID)
		}
	}

	return s.repo.FindPaginated(ctx, page, limit, searchTerm, ids)
}

func (s *DescargoService) Update(ctx context.Context, descargo *models.Descargo) error {
	return s.repo.Update(ctx, descargo)
}

func (s *DescargoService) Submit(ctx context.Context, id, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado && descargo.Estado != models.EstadoDescargoOpenTicket {
		return fmt.Errorf("el descargo no se puede enviar en su estado actual (%s)", descargo.Estado)
	}

	oldState := descargo.Estado
	descargo.Estado = models.EstadoDescargoEnRevision
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "ENVIAR_DESCARGO", "descargo", id, string(oldState), string(models.EstadoDescargoEnRevision), "", "")
	slog.Info("Descargo enviado a revisión", "id", id, "codigo", descargo.Codigo, "user_id", userID)
	return nil
}

func (s *DescargoService) Approve(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision {
		return errors.New("solo se pueden aprobar descargos en revisión")
	}

	hasOpenTickets := false
	for _, tramo := range descargo.Tramos {
		if tramo.EsOpenTicket {
			hasOpenTickets = true
			break
		}
	}

	newState := models.EstadoDescargoFinalizado
	if hasOpenTickets {
		newState = models.EstadoDescargoOpenTicket
	}

	descargo.Estado = newState
	descargo.Observaciones = ""
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "APROBAR_DESCARGO", "descargo", id, string(models.EstadoDescargoEnRevision), string(newState), "", "")

	if hasOpenTickets {
		slog.Info("Descargo aprobado parcialmente (Open Tickets pendientes)", "id", id, "codigo", descargo.Codigo, "user_id", userID)
	} else {
		slog.Info("Descargo aprobado y finalizado", "id", id, "codigo", descargo.Codigo, "user_id", userID)
	}

	// Only finalize the parent Solicitud if the Descargo is fully finalized (no Open Tickets left)
	if !hasOpenTickets && descargo.SolicitudID != "" {
		return s.solicitudService.Finalize(ctx, descargo.SolicitudID)
	}

	return nil
}

func (s *DescargoService) Reject(ctx context.Context, id string, userID string, observaciones string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision {
		return errors.New("solo se pueden observar descargos en revisión")
	}

	descargo.Estado = models.EstadoDescargoRechazado
	descargo.Observaciones = observaciones
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "RECHAZAR_DESCARGO", "descargo", id, string(models.EstadoDescargoEnRevision), string(models.EstadoDescargoRechazado), "", "")
	slog.Info("Descargo rechazado (observado)", "id", id, "codigo", descargo.Codigo, "user_id", userID, "observaciones", observaciones)
	return nil
}

func (s *DescargoService) RevertToDraft(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoFinalizado && descargo.Estado != models.EstadoDescargoOpenTicket {
		return errors.New("solo se puede revertir un descargo finalizado o en espera")
	}

	// 1. Manejo de Créditos vinculados (No permitir si ya se usaron)
	creditos, err := s.openTicketRepo.FindByDescargoID(ctx, id)
	if err == nil && len(creditos) > 0 {
		for _, c := range creditos {
			if c.Estado == models.EstadoOpenTicketFinalizado {
				return fmt.Errorf("no se puede revertir: el crédito de viaje generado ya fue utilizado")
			}
		}
		// Eliminar créditos generados (solo si no se usaron, verificado arriba)
		for _, c := range creditos {
			if err := s.openTicketRepo.Delete(ctx, c.ID); err != nil {
				slog.Error("Error eliminado Open Ticket tras reversión", "error", err, "ticket_id", c.ID)
			}
		}
	}

	descargo.Estado = models.EstadoDescargoBorrador
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo revertido a borrador", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	if descargo.SolicitudID != "" {
		return s.solicitudService.RevertFinalize(ctx, descargo.SolicitudID)
	}

	return nil
}

// SyncOpenTickets sincroniza los tramos marcados como 'Open Ticket' con la tabla de open_tickets.
func (s *DescargoService) SyncOpenTickets(ctx context.Context, descargoID string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, descargoID)
	if err != nil {
		return err
	}

	// 0. Seguridad: No se generan Open Tickets para pasajes oficiales (Comisiones/Misiones)
	if descargo.Solicitud != nil && descargo.Solicitud.GetConceptoCodigo() == "OFICIAL" {
		return nil
	}

	// 1. Obtener registros existentes para este descargo
	existing, _ := s.openTicketRepo.FindByDescargoID(ctx, descargoID)
	existingMap := make(map[string]models.OpenTicket)
	for _, t := range existing {
		existingMap[t.NumeroBillete] = t
	}

	// 2. Identificar tramos marcados como OpenTicket en el itinerario
	openTicketsInDescargo := make(map[string]models.OpenTicket)
	for _, tramo := range descargo.Tramos {
		if tramo.EsOpenTicket && tramo.Billete != "" {
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
					// Fallback: Buscar el pasaje por número de billete si no viene en el tramo o es vacío
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
	}

	// 3. Persistir cambios
	for num, ticket := range openTicketsInDescargo {
		if old, ok := existingMap[num]; ok {
			// Actualizar si está pendiente o disponible
			if old.Estado == models.EstadoOpenTicketPendiente || old.Estado == models.EstadoOpenTicketDisponible {
				old.TramosNoUsados = ticket.TramosNoUsados
				old.PasajeID = ticket.PasajeID
				old.UpdatedBy = &userID
				_ = s.openTicketRepo.Update(ctx, &old)
			}
			delete(existingMap, num)
		} else {
			// Crear nuevo registro
			ticket.CreatedBy = &userID
			_ = s.openTicketRepo.Create(ctx, &ticket)
		}
	}

	// 4. Eliminar lo que ya no está marcado (solo si sigue pendiente)
	for _, old := range existingMap {
		if old.Estado == models.EstadoOpenTicketPendiente {
			_ = s.openTicketRepo.Delete(ctx, old.ID)
		}
	}

	return nil
}
