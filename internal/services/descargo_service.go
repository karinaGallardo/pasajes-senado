package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"time"

	"gorm.io/gorm"
)

type DescargoService struct {
	repo             *repositories.DescargoRepository
	pasajeRepo       *repositories.PasajeRepository
	solicitudService *SolicitudService
	usuarioService   *UsuarioService
	auditService     *AuditService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	pasajeRepo *repositories.PasajeRepository,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
	auditService *AuditService,
) *DescargoService {
	return &DescargoService{
		repo:             repo,
		pasajeRepo:       pasajeRepo,
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

func (s *DescargoService) Liquidate(ctx context.Context, id string, pasajeIDs []string, pasajeCostos []float64, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoAprobado {
		return errors.New("solo se pueden liquidar descargos que ya han sido aprobados técnicamente")
	}

	return s.repo.RunTransaction(func(repo *repositories.DescargoRepository, tx *gorm.DB) error {
		txPasajeRepo := s.pasajeRepo.WithTx(tx)
		totalDevolucion := 0.0

		// 1. Actualizar pasajes individualmente y sumar diferencias
		for i, pID := range pasajeIDs {
			if i >= len(pasajeCostos) {
				continue
			}

			pasaje, err := txPasajeRepo.FindByID(ctx, pID)
			if err != nil {
				continue
			}

			pasaje.CostoUtilizacion = pasajeCostos[i]
			pasaje.Diferencia = pasaje.Costo - pasaje.CostoUtilizacion
			totalDevolucion += pasaje.Diferencia

			if err := txPasajeRepo.Update(ctx, pasaje); err != nil {
				return fmt.Errorf("error actualizando pasaje %s: %w", pID, err)
			}
		}

		// 2. Actualizar Descargo
		oldMonto := descargo.MontoDevolucion
		descargo.MontoDevolucion = totalDevolucion

		if totalDevolucion > 0 {
			descargo.Estado = models.EstadoDescargoPendientePago
		} else {
			descargo.Estado = models.EstadoDescargoFinalizado
		}
		descargo.UpdatedBy = &userID

		if err := repo.Update(ctx, descargo); err != nil {
			return err
		}

		s.auditService.Log(ctx, "LIQUIDAR_DESCARGO", "descargo", id, string(models.EstadoDescargoAprobado), string(descargo.Estado), fmt.Sprintf("Monto: %.2f", totalDevolucion), fmt.Sprintf("Anterior: %.2f", oldMonto))
		slog.Info("Descargo liquidado financieramente con desglose", "id", id, "codigo", descargo.Codigo, "monto", totalDevolucion, "user_id", userID)

		return nil
	})
}

func (s *DescargoService) RevertLiquidation(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Permiso según el modelo
	adminRol := models.RolAdmin
	if !descargo.CanRevertLiquidation(&models.Usuario{BaseModel: models.BaseModel{ID: userID}, RolCodigo: &adminRol}) {
		return fmt.Errorf("no tiene permiso para revertir esta liquidación")
	}

	if descargo.Estado != models.EstadoDescargoPendientePago {
		return errors.New("solo se pueden revertir liquidaciones que están en estado PENDIENTE_PAGO")
	}

	descargo.Estado = models.EstadoDescargoAprobado
	descargo.UpdatedBy = &userID

	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "REVERTIR_LIQUIDACION", "descargo", id, string(models.EstadoDescargoPendientePago), string(models.EstadoDescargoAprobado), "Corrección de montos", "")
	slog.Info("Liquidación revertida para corrección", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	return nil
}

func (s *DescargoService) ReportPayment(ctx context.Context, id string, filePath string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoPendientePago {
		return errors.New("solo se puede reportar pago para descargos en estado PENDIENTE_PAGO")
	}

	now := time.Now()
	descargo.ComprobantePago = filePath
	descargo.FechaPago = &now
	descargo.Estado = models.EstadoDescargoPagado
	descargo.UpdatedBy = &userID

	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "REPORTAR_PAGO", "descargo", id, string(models.EstadoDescargoPendientePago), string(models.EstadoDescargoPagado), filePath, "")
	slog.Info("Pago de descargo reportado", "id", id, "codigo", descargo.Codigo, "archivo", filePath, "user_id", userID)

	return nil
}

func (s *DescargoService) Finalize(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoPagado {
		return errors.New("solo se pueden finalizar descargos cuyo pago ha sido reportado")
	}

	descargo.Estado = models.EstadoDescargoFinalizado
	descargo.UpdatedBy = &userID

	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "FINALIZAR_DESCARGO", "descargo", id, string(models.EstadoDescargoPagado), string(models.EstadoDescargoFinalizado), "", "")
	slog.Info("Descargo finalizado oficialmente", "id", id, "codigo", descargo.Codigo, "user_id", userID)

	return nil
}

func (s *DescargoService) RevertPayment(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	// Permiso según el modelo
	adminRol := models.RolAdmin
	if !descargo.CanRevertPayment(&models.Usuario{BaseModel: models.BaseModel{ID: userID}, RolCodigo: &adminRol}) {
		return fmt.Errorf("no tiene permiso para observar este pago")
	}

	oldEstado := descargo.Estado
	descargo.Estado = models.EstadoDescargoPendientePago
	descargo.UpdatedBy = &userID

	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "AVISO_PAGO_INCORRECTO", "descargo", id, string(oldEstado), string(descargo.Estado), "Reversión por comprobante incorrecto", "")
	return nil
}

func (s *DescargoService) RevertFinalization(ctx context.Context, id string, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoFinalizado {
		return errors.New("solo se pueden revertir descargos en estado FINALIZADO")
	}

	oldEstado := descargo.Estado
	descargo.Estado = models.EstadoDescargoAprobado
	descargo.UpdatedBy = &userID

	if err := s.repo.Update(ctx, descargo); err != nil {
		return err
	}

	s.auditService.Log(ctx, "REVERTIR_FINALIZACION", "descargo", id, string(oldEstado), string(descargo.Estado), "Reapertura para rectificación de datos", "")
	return nil
}
