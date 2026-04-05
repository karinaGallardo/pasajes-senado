package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"sort"
	"strings"
)

type DescargoService struct {
	repo             *repositories.DescargoRepository
	solicitudService *SolicitudService
	usuarioService   *UsuarioService
	auditService     *AuditService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
	auditService *AuditService,
) *DescargoService {
	return &DescargoService{
		repo:             repo,
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

	if descargo.Estado != models.EstadoDescargoBorrador && descargo.Estado != models.EstadoDescargoRechazado {
		return fmt.Errorf("el descargo no se puede enviar en su estado actual (%s)", descargo.Estado)
	}

	descargo.Estado = models.EstadoDescargoEnRevision
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "ENVIAR_DESCARGO", "descargo", id, string(models.EstadoDescargoBorrador), string(models.EstadoDescargoEnRevision), "", "")
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

	descargo.Estado = models.EstadoDescargoAprobado
	descargo.Observaciones = ""
	descargo.UpdatedBy = &userID
	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "APROBAR_DESCARGO", "descargo", id, string(models.EstadoDescargoEnRevision), string(models.EstadoDescargoAprobado), "", "")
	slog.Info("Descargo aprobado", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	if descargo.SolicitudID != "" {
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

	if descargo.Estado != models.EstadoDescargoAprobado {
		return errors.New("solo se puede revertir un descargo aprobado")
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

func (s *DescargoService) GetItinerarioParaDescargo(ctx context.Context, solicitud *models.Solicitud) (map[string][]dtos.TramoView, map[string][]dtos.TramoView) {
	pasajesOriginales := make(map[string][]dtos.TramoView)
	pasajesReprogramados := make(map[string][]dtos.TramoView)

	s.processTicketItemsForView(solicitud.GetItemIda(), pasajesOriginales, pasajesReprogramados)
	s.processTicketItemsForView(solicitud.GetItemVuelta(), pasajesOriginales, pasajesReprogramados)

	return pasajesOriginales, pasajesReprogramados
}

func (s *DescargoService) processTicketItemsForView(item *models.SolicitudItem, targetOrig, targetRepro map[string][]dtos.TramoView) {
	if item == nil {
		return
	}

	tipo := string(item.Tipo)
	pasajes := item.Pasajes

	sort.Slice(pasajes, func(i, j int) bool {
		if pasajes[i].Seq != pasajes[j].Seq {
			return pasajes[i].Seq < pasajes[j].Seq
		}
		return pasajes[i].CreatedAt.Before(pasajes[j].CreatedAt)
	})

	for _, p := range pasajes {
		if !p.IsDischargeable() {
			continue
		}

		targetMap := targetOrig
		// Se eliminó la clasificación por PasajeAnteriorID ya que ahora se revierte y edita el pasaje original

		tramosVuelo := p.GetTramosRuta()
		for _, seg := range tramosVuelo {
			parts := strings.Split(seg, " - ")
			rv := dtos.RutaView{Display: seg}
			if len(parts) == 2 {
				rv.Origen = parts[0]
				rv.Destino = parts[1]
			} else {
				rv.Origen = seg
			}

			targetMap[tipo] = append(targetMap[tipo], dtos.TramoView{
				ID:              p.ID,
				Ruta:            rv,
				RutaID:          utils.DerefString(p.RutaID),
				Fecha:           p.FechaVuelo.Format("2006-01-02"),
				Billete:         p.NumeroBillete,
				EsDevolucion:    false,
				EsModificacion:  false,
				MontoDevolucion: 0,
			})
		}
	}
}
