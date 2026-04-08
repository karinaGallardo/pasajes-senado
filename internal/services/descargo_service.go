package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"gorm.io/gorm"
)

type DescargoService struct {
	repo             *repositories.DescargoRepository
	pasajeRepo       *repositories.PasajeRepository
	creditoRepo      *repositories.CreditoPasajeRepository
	solicitudService *SolicitudService
	usuarioService   *UsuarioService
	auditService     *AuditService
}

func NewDescargoService(
	repo *repositories.DescargoRepository,
	pasajeRepo *repositories.PasajeRepository,
	creditoRepo *repositories.CreditoPasajeRepository,
	solicitudService *SolicitudService,
	usuarioService *UsuarioService,
	auditService *AuditService,
) *DescargoService {
	return &DescargoService{
		repo:             repo,
		pasajeRepo:       pasajeRepo,
		creditoRepo:      creditoRepo,
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

func (s *DescargoService) Liquidate(ctx context.Context, id string, pasajeIDs []string, pasajeCostos []float64, montosCredito []float64, montosDevolucion []float64, userID string) error {
	descargo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if descargo.Estado != models.EstadoDescargoAprobado {
		return errors.New("solo se pueden liquidar descargos que ya han sido aprobados técnicamente")
	}

	return s.repo.RunTransaction(func(repo *repositories.DescargoRepository, tx *gorm.DB) error {
		txPasajeRepo := s.pasajeRepo.WithTx(tx)
		txCreditoRepo := s.creditoRepo.WithTx(tx)

		totalCreditoGral := 0.0
		totalDevolucionGral := 0.0

		var rutasCredito []string
		var billetesCredito []string

		// 1. Procesar pasajes uno a uno
		for i, pID := range pasajeIDs {
			pasaje, err := txPasajeRepo.FindByID(ctx, pID)
			if err != nil {
				continue
			}

			// Tomar los montos de ahorro del request o 0
			mCredito := 0.0
			if i < len(montosCredito) {
				mCredito = montosCredito[i]
			}
			mDevolucion := 0.0
			if i < len(montosDevolucion) {
				mDevolucion = montosDevolucion[i]
			}

			// El costo utilizado es el original menos los ahorros
			pasaje.CostoUtilizado = pasaje.Costo - mCredito - mDevolucion
			pasaje.MontoCredito = mCredito
			pasaje.MontoReembolso = mDevolucion

			if err := txPasajeRepo.Update(ctx, pasaje); err != nil {
				return fmt.Errorf("error actualizando pasaje %s: %w", pID, err)
			}

			// Acumular ahorros para el total del descargo
			if mCredito > 0 {
				totalCreditoGral += mCredito

				// Solo armar la ruta con los tramos que son DEVOLUCIÓN
				var tramosDev []string
				for _, t := range pasaje.DescargoTramos {
					if t.EsDevolucion {
						tramosDev = append(tramosDev, t.GetRutaDisplay())
					}
				}
				if len(tramosDev) > 0 {
					rutasCredito = append(rutasCredito, strings.Join(tramosDev, " ; "))
				} else {
					// Fallback por si no hay tramos marcados pero hay monto
					rutasCredito = append(rutasCredito, pasaje.GetRutaDisplay())
				}

				if pasaje.NumeroBillete != "" {
					billetesCredito = append(billetesCredito, pasaje.NumeroBillete)
				}
			}
			if mDevolucion > 0 {
				totalDevolucionGral += mDevolucion
			}
		}

		// 2. Gestionar el crédito consolidado de forma idempotente
		// Buscamos explícitamente por el ID del descargo en la transacción actual
		existingCreditos, err := txCreditoRepo.FindByDescargoID(ctx, id)
		if err != nil {
			slog.Error("Error buscando créditos previos", "descargo_id", id, "error", err)
		}

		if totalCreditoGral > 0 {
			if len(existingCreditos) > 0 {
				// Actualizar el primero (referencia directa al slice)
				c := &existingCreditos[0]
				c.Monto = totalCreditoGral
				c.RutaReferencia = utils.UniqueStringsJoin(rutasCredito, ", ")
				c.BilletesReferencia = utils.UniqueStringsJoin(billetesCredito, ", ")
				c.Observaciones = fmt.Sprintf("Liquidación actualizada. Rutas: %s", c.RutaReferencia)
				c.UpdatedBy = &userID

				if err := txCreditoRepo.Update(ctx, c); err != nil {
					return fmt.Errorf("error actualizando crédito: %w", err)
				}

				// Si por algún error previo hubieran más, los limpiamos (idempotencia agresiva)
				if len(existingCreditos) > 1 {
					for i := 1; i < len(existingCreditos); i++ {
						_ = txCreditoRepo.Delete(ctx, existingCreditos[i].ID)
					}
				}
			} else {
				// Crear uno nuevo
				nuevoCredito := &models.CreditoPasaje{
					UsuarioID:          descargo.UsuarioID,
					DescargoID:         descargo.ID,
					Monto:              totalCreditoGral,
					RutaReferencia:     utils.UniqueStringsJoin(rutasCredito, ", "),
					BilletesReferencia: utils.UniqueStringsJoin(billetesCredito, ", "),
					Estado:             models.EstadoCreditoPendiente,
					CreatedBy:          &userID,
					Observaciones:      fmt.Sprintf("Crédito generado en liquidación de descargo %s (esperando finalización)", descargo.Codigo),
				}
				if err := txCreditoRepo.Create(ctx, nuevoCredito); err != nil {
					return fmt.Errorf("error creando crédito de pasaje: %w", err)
				}
			}
		} else {
			// Si el monto es 0, borrar cualquier crédito previo
			for _, c := range existingCreditos {
				if c.Estado == models.EstadoCreditoUsado {
					return fmt.Errorf("no se puede anular el crédito: ya fue utilizado")
				}
				_ = txCreditoRepo.Delete(ctx, c.ID)
			}
		}

		// 3. Actualizar Descargo
		// El monto de devolución del descargo es lo que el usuario DEBE DEVOLVER al banco/senado
		oldMonto := descargo.MontoDevolucion
		descargo.MontoDevolucion = totalDevolucionGral

		if descargo.MontoDevolucion > 0 {
			descargo.Estado = models.EstadoDescargoPendientePago
		} else {
			descargo.Estado = models.EstadoDescargoFinalizado
		}

		descargo.UpdatedBy = &userID

		if err := repo.Update(ctx, descargo); err != nil {
			return err
		}

		s.auditService.Log(ctx, "LIQUIDAR_DESCARGO", "descargo", id, string(models.EstadoDescargoAprobado), string(descargo.Estado),
			fmt.Sprintf("Efectivo: %.2f, Crédito: %.2f", totalDevolucionGral, totalCreditoGral),
			fmt.Sprintf("Anterior Efectivo: %.2f", oldMonto))

		slog.Info("Descargo liquidado con desglose", "id", id, "efectivo", totalDevolucionGral, "credito", totalCreditoGral)

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

	// 1. Manejo de Créditos vinculados
	creditos, err := s.creditoRepo.FindByDescargoID(ctx, id)
	if err == nil && len(creditos) > 0 {
		for _, c := range creditos {
			if c.Estado == models.EstadoCreditoUsado {
				return fmt.Errorf("no se puede corregir la liquidación: el crédito ya fue utilizado")
			}
			// Simplemente nos aseguramos que estén en PENDIENTE mientras se corrige
			if c.Estado != models.EstadoCreditoPendiente {
				c.Estado = models.EstadoCreditoPendiente
				c.Observaciones += " | Regresado a pendiente por corrección de liquidación."
				_ = s.creditoRepo.Update(ctx, &c)
			}
		}
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

	// 2. Activar créditos de viaje ahora que el descargo está oficialmente cerrado
	creditos, err := s.creditoRepo.FindByDescargoID(ctx, id)
	if err == nil {
		for _, c := range creditos {
			if c.Estado == models.EstadoCreditoPendiente {
				c.Estado = models.EstadoCreditoDisponible
				c.Observaciones += " | Activado por finalización de descargo."
				c.UpdatedBy = &userID
				_ = s.creditoRepo.Update(ctx, &c)
			}
		}
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

	// 1. Manejo de Créditos vinculados (Especialmente para Derecho)
	creditos, err := s.creditoRepo.FindByDescargoID(ctx, id)
	if err == nil && len(creditos) > 0 {
		for _, c := range creditos {
			if c.Estado == models.EstadoCreditoUsado {
				return fmt.Errorf("no se puede revertir el finalizado: el crédito de viaje generado ya fue utilizado")
			}
		}
		// Ponemos los créditos en PENDIENTE nuevamente para evitar que se usen mientras se corrige
		for _, c := range creditos {
			if c.Estado == models.EstadoCreditoDisponible {
				c.Estado = models.EstadoCreditoPendiente
				c.Observaciones += " | Suspendido por reversión de finalización."
				_ = s.creditoRepo.Update(ctx, &c)
			}
		}
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
