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
	repo              *repositories.DescargoRepository
	pasajeRepo        *repositories.PasajeRepository
	openTicketService *OpenTicketService
	solicitudService  *SolicitudService
	usuarioService    *UsuarioService
	auditService      *AuditService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	pasajeRepo *repositories.PasajeRepository,
	openTicketService *OpenTicketService,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
	auditService *AuditService,
) *DescargoService {
	return &DescargoService{
		repo:              repo,
		pasajeRepo:        pasajeRepo,
		openTicketService: openTicketService,
		solicitudService:  solicitudService,
		usuarioService:    usuarioService,
		auditService:      auditService,
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
	newState := models.EstadoDescargoEnRevision

	descargo.Estado = newState
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	// Sincronizar Open Tickets desde los tramos marcados
	_ = s.openTicketService.SyncFromDescargo(ctx, descargo, userID)

	s.auditService.Log(ctx, "ENVIAR_DESCARGO", "descargo", id, string(oldState), string(newState), "", "")
	slog.Info("Descargo enviado a revisión", "id", id, "codigo", descargo.Codigo, "user_id", userID, "state", newState)
	return nil
}

func (s *DescargoService) Approve(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision && descargo.Estado != models.EstadoDescargoEnRevisionOT {
		return errors.New("solo se pueden aprobar descargos en revisión")
	}

	// Todos los descargos se finalizan — los OTs generados son créditos independientes.
	// Ya no se deja el descargo en estado OPEN_TICKET esperando reprogramación.
	newState := models.EstadoDescargoFinalizado

	descargo.Estado = newState
	descargo.Observaciones = ""
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "APROBAR_DESCARGO", "descargo", id, string(models.EstadoDescargoEnRevision), string(newState), "", "")

	slog.Info("Descargo aprobado y finalizado", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	// Auto-aprobar los Open Tickets generados desde este descargo
	user, _ := s.usuarioService.GetByID(ctx, userID)
	if ots, _ := s.openTicketService.DescargoOTs(ctx, descargo.ID); len(ots) > 0 {
		for _, ot := range ots {
			if ot.Estado == models.EstadoOpenTicketPendiente {
				_ = s.openTicketService.Approve(ctx, ot.ID, user)
			}
		}
	}

	// Finalizar la solicitud padre
	if descargo.SolicitudID != "" {
		return s.solicitudService.Finalize(ctx, descargo.SolicitudID, user)
	}

	return nil
}

func (s *DescargoService) Reject(ctx context.Context, id string, userID string, observaciones string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoEnRevision && descargo.Estado != models.EstadoDescargoEnRevisionOT {
		return errors.New("solo se pueden observar descargos en revisión")
	}

	// Lógica de Rechazo Escalonado
	targetState := models.EstadoDescargoRechazado
	if descargo.Estado == models.EstadoDescargoEnRevisionOT {
		targetState = models.EstadoDescargoRechazadoOT
	}

	descargo.Estado = targetState
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
	if creditos, err := s.openTicketService.DescargoOTs(ctx, id); err == nil && len(creditos) > 0 {
		for _, c := range creditos {
			if c.Estado == models.EstadoOpenTicketFinalizado {
				return fmt.Errorf("no se puede revertir: el crédito de viaje generado ya fue utilizado")
			}
		}
		if err := s.openTicketService.DeletePending(ctx, id); err != nil {
			slog.Error("Error eliminado Open Ticket tras reversión", "error", err, "descargo_id", id)
		}
	}

	// Lógica de Reversión Escalonada
	targetState := models.EstadoDescargoBorrador
	if descargo.Estado == models.EstadoDescargoFinalizado && descargo.HasOpenTicket() {
		targetState = models.EstadoDescargoOpenTicket
	}

	descargo.Estado = targetState
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	slog.Info("Descargo revertido", "id", id, "nuevo_estado", targetState, "user_id", userID)

	// Solo revertir la solicitud si volvemos a BORRADOR
	if targetState == models.EstadoDescargoBorrador && descargo.SolicitudID != "" {
		user, _ := s.usuarioService.GetByID(ctx, userID)
		return s.solicitudService.RevertFinalize(ctx, descargo.SolicitudID, user)
	}

	return nil
}
