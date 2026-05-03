package services

import (
	"context"
	"errors"
	"fmt"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func NewSolicitudService(
	repo *repositories.SolicitudRepository,
	itemRepo *repositories.CupoDerechoItemRepository,
	solicitudItemRepo *repositories.SolicitudItemRepository,
	emailService *EmailService,
	auditService *AuditService,
	openTicketRepo *repositories.OpenTicketRepository,
) *SolicitudService {
	return &SolicitudService{
		repo:              repo,
		itemRepo:          itemRepo,
		solicitudItemRepo: solicitudItemRepo,
		emailService:      emailService,
		auditService:      auditService,
		openTicketRepo:    openTicketRepo,
	}
}

type SolicitudService struct {
	repo              *repositories.SolicitudRepository
	itemRepo          *repositories.CupoDerechoItemRepository
	solicitudItemRepo *repositories.SolicitudItemRepository
	emailService      *EmailService
	auditService      *AuditService
	openTicketRepo    *repositories.OpenTicketRepository
}

// CreateDerecho and CreateOficial moved to specialized services.

// Helper locally if not in utils

func (s *SolicitudService) sendCreationEmail(solicitud *models.Solicitud) {
	fullSol, err := s.GetByID(context.Background(), solicitud.ID)
	if err != nil {
		fmt.Printf("Error reloading solicitud for email: %v\n", err)
		return
	}

	beneficiary := fullSol.Usuario
	if beneficiary.Email == "" {
		fmt.Printf("Beneficiary %s has no email\n", beneficiary.Username)
		return
	}

	concepto := fullSol.GetConceptoNombre()
	if concepto == "" {
		concepto = "PASAJES"
	}

	subject := fmt.Sprintf("[%s] Solicitud de Pasaje Creada - %s", strings.ToUpper(concepto), fullSol.Codigo)

	tramosHTML := ""
	for _, it := range fullSol.Items {
		origen := it.OrigenIATA
		if it.Origen != nil {
			origen = it.Origen.Ciudad
		}
		destino := it.DestinoIATA
		if it.Destino != nil {
			destino = it.Destino.Ciudad
		}
		fechaIt := "-"
		if it.Fecha != nil {
			fechaIt = utils.FormatDateShortES(*it.Fecha)
		}
		tramosHTML += fmt.Sprintf(`
			<tr>
				<td style="padding: 8px; border-bottom: 1px solid #eee; width: 30%%;"><strong>Tramo %s:</strong></td>
				<td style="padding: 8px; border-bottom: 1px solid #eee;">%s &rarr; %s (%s)</td>
			</tr>`, it.Tipo, origen, destino, fechaIt)
	}

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solPath := "derecho"
	if fullSol.IsOficial() {
		solPath = "oficial"
	}
	solURL := fmt.Sprintf("%s/solicitudes/%s/%s/detalle", baseURL, solPath, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #03738C; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Solicitud de Pasaje Registrada</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Concepto: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>Se ha registrado una nueva solicitud de pasaje a su nombre.</p>

				<h3 style="color: #03738C; margin-bottom: 10px;">Información General</h3>
				<table style="border-collapse: collapse; width: 100%%; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee; width: 30%%;"><strong>Código:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Concepto:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Estado:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
				</table>

				<h3 style="color: #03738C; margin-bottom: 10px;">Itinerario Solicitado</h3>
				<table style="border-collapse: collapse; width: 100%%; margin-bottom: 20px;">
					%s
				</table>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #03738C; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Detalles en el Sistema</a>
				</div>
				<p style="font-size: 11px; color: #999; margin-top: 20px; text-align: center;">
					Si el botón no funciona/no se ve, copie esta URL:<br>
					%s
				</p>
			</div>
		</div>
	`, strings.ToUpper(concepto), beneficiary.GetNombreCompleto(), fullSol.Codigo, concepto, fullSol.GetEstado(), tramosHTML, solURL, solURL)

	err = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
	}
}

func (s *SolicitudService) sendRevertApprovalEmail(solicitud *models.Solicitud) {
	fullSol, err := s.GetByID(context.Background(), solicitud.ID)
	if err != nil {
		fmt.Printf("Error reloading solicitud for email: %v\n", err)
		return
	}

	beneficiary := fullSol.Usuario
	if beneficiary.Email == "" {
		return
	}

	concepto := fullSol.GetConceptoNombre()
	if concepto == "" {
		concepto = "PASAJES"
	}

	subject := fmt.Sprintf("[%s] Solicitud de Pasaje Revertida - %s", strings.ToUpper(concepto), fullSol.Codigo)

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solPath := "derecho"
	if fullSol.IsOficial() {
		solPath = "oficial"
	}
	solURL := fmt.Sprintf("%s/solicitudes/%s/%s/detalle", baseURL, solPath, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #d97706; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Aprobación Revertida</h2>
				<p style="margin: 5px 0 0 0; opacity: 0.9;">Concepto: %s</p>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>La aprobación de su solicitud <strong>%s</strong> ha sido revertida al estado <strong>SOLICITADO</strong>.</p>
				<p>Concepto de la solicitud: <strong>%s</strong></p>
				<p>Esto puede deberse a ajustes necesarios en el itinerario o cambios de último momento.</p>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #d97706; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Solicitud</a>
				</div>
				<p style="font-size: 11px; color: #999; margin-top: 20px; text-align: center;">
					Si el botón no funciona/no se ve, copie esta URL:<br>
					%s
				</p>
			</div>
		</div>
	`, strings.ToUpper(concepto), beneficiary.GetNombreCompleto(), fullSol.Codigo, concepto, solURL, solURL)

	_ = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
}

func (s *SolicitudService) GetAll(ctx context.Context, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindAll(ctx, status, concepto)
}

func (s *SolicitudService) GetByUserID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindByUserID(ctx, userID, status, concepto)
}

func (s *SolicitudService) GetByUserIdOrAccesibleByEncargadoID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	return s.repo.FindByUserIdOrAccesibleByEncargadoID(ctx, userID, status, concepto)
}

func (s *SolicitudService) GetPaginated(ctx context.Context, userID string, isAdmin bool, status string, concepto string, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindPaginated(ctx, userID, isAdmin, status, concepto, page, limit, searchTerm)
}

func (s *SolicitudService) GetPendientesDescargo(ctx context.Context, userID string, isAdmin bool) ([]models.Solicitud, error) {
	return s.repo.FindPendientesDeDescargoUI(ctx, userID, isAdmin)
}

func (s *SolicitudService) GetPendientesDescargoPaginated(ctx context.Context, userID string, isAdmin bool, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindPendientesDeDescargoPaginated(ctx, userID, isAdmin, page, limit, searchTerm)
}

func (s *SolicitudService) GetConDescargoOpenTicketPaginated(ctx context.Context, userID string, isAdmin bool, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindWithOpenTicketDescargoPaginated(ctx, userID, isAdmin, page, limit, searchTerm)
}

func (s *SolicitudService) GetConDescargoOpenTicketCount(ctx context.Context, userID string, isAdmin bool) int64 {
	count, _ := s.repo.CountWithOpenTicketDescargo(ctx, userID, isAdmin)
	return count
}

func (s *SolicitudService) GetEnRevisionDescargoPaginated(ctx context.Context, userID string, isAdmin bool, page, limit int, searchTerm string) (*repositories.PaginatedSolicitudes, error) {
	return s.repo.FindEnRevisionDescargoPaginated(ctx, userID, isAdmin, page, limit, searchTerm)
}

func (s *SolicitudService) GetEnRevisionDescargoCount(ctx context.Context, userID string, isAdmin bool) int64 {
	count, _ := s.repo.CountEnRevisionDescargo(ctx, userID, isAdmin)
	return count
}

func (s *SolicitudService) GetByCupoDerechoItemID(ctx context.Context, itemID string) ([]models.Solicitud, error) {
	return s.repo.FindByCupoDerechoItemID(ctx, itemID)
}

func (s *SolicitudService) GetByID(ctx context.Context, id string) (*models.Solicitud, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *SolicitudService) GetItemByID(ctx context.Context, id string) (*models.SolicitudItem, error) {
	return s.solicitudItemRepo.FindByID(ctx, id)
}

func (s *SolicitudService) Approve(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		if err := solicitud.Approve(); err != nil {
			return err
		}

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return s.auditService.Log(ctx, "APROBAR_SOLICITUD", "solicitud", solicitud.ID, "SOLICITADO", "APROBADO", "", "")
	})
}

func (s *SolicitudService) RevertApproval(ctx context.Context, id string) error {
	var solicitudForEmail *models.Solicitud
	err := s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		stOld := solicitud.GetEstado()

		if err := solicitud.RevertApproval(); err != nil {
			return err
		}

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}
		solicitudForEmail = solicitud
		return s.auditService.Log(ctx, "REVERTIR_APROBACION", "solicitud", solicitud.ID, stOld, "SOLICITADO", "", "")
	})

	if err == nil && solicitudForEmail != nil {
		go s.sendRevertApprovalEmail(solicitudForEmail)
	}

	return err
}

func (s *SolicitudService) Finalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// Mark all items as FINALIZADO and their pasajes as USADO via Model
		if err := solicitud.Finalize(); err != nil {
			return err
		}

		if err := tx.Save(solicitud).Error; err != nil {
			return err
		}
		if err := tx.Save(solicitud.Items).Error; err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "USADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) Reject(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// Reject all items via Model
		if err := solicitud.Reject(); err != nil {
			return err
		}

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return s.auditService.Log(ctx, "RECHAZAR_SOLICITUD", "solicitud", solicitud.ID, "SOLICITADO", "RECHAZADO", "", "")
	})
}

func (s *SolicitudService) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Obtener el estado ACTUAL de la DB
		var dbSol models.Solicitud
		if err := tx.Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).First(&dbSol, "id = ?", solicitud.ID).Error; err != nil {
			return err
		}

		dbItems := make(map[string]models.SolicitudItem)
		for _, it := range dbSol.Items {
			dbItems[it.ID] = it
		}

		// 2. Procesar tramos
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			old, exists := dbItems[item.ID]

			if !exists {
				fmt.Printf("DEBUG: Tramo nuevo detectado ID=%s\n", item.ID)
				if err := tx.Save(item).Error; err != nil {
					return err
				}
				continue
			}

			changes := item.GetChanges(old)
			if len(changes) > 0 {
				fmt.Printf("DEBUG: Cambios en Tramo %s (%s): %v\n", item.ID, item.Tipo, changes)
				if err := tx.Model(item).Updates(changes).Error; err != nil {
					return err
				}
			}
		}

		// 3. Actualizar Solicitud (Padre)
		// UpdateStatusBasedOnItems() se ejecuta vía Hooks GORM al guardar/actualizar
		parentChanges := solicitud.GetChanges(dbSol)

		if len(parentChanges) > 0 {
			fmt.Printf("DEBUG: Cambios en Solicitud %s: %v\n", solicitud.Codigo, parentChanges)
			if err := tx.Model(solicitud).Updates(parentChanges).Error; err != nil {
				return err
			}
		} else {
			fmt.Printf("DEBUG: No se detectaron cambios en Solicitud %s\n", solicitud.Codigo)
		}

		return nil
	})
}

func (s *SolicitudService) Delete(ctx context.Context, id string, deletedBy string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		if !solicitud.IsDeletableState() {
			return errors.New("solo se pueden eliminar solicitudes en estado SOLICITADO. El estado actual es: " + solicitud.GetEstado())
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return repoTx.Delete(ctx, id, deletedBy)
	})
}

func (s *SolicitudService) ApproveItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}

		if !solicitud.ApproveItem(itemID) {
			return fmt.Errorf("item no encontrado")
		}

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		// Si es derecho, marcar como reservado
		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				itemRepoTx.Update(ctx, item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RejectItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}

		if !solicitud.RejectItem(itemID) {
			return fmt.Errorf("item no encontrado")
		}

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}

		// Si todos los items activos son rechazados, liberar el cupo
		if solicitud.AreAllItemsInactive() && solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				itemRepoTx.Update(ctx, item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RevertApprovalItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Verificar si existe la solicitud
		sol, err := repoTx.FindByID(ctx, solicitudID)
		if err != nil {
			return err
		}

		// 2. Verificar si el tramo (item) existe
		item := sol.GetItemByID(itemID)
		if item == nil {
			return fmt.Errorf("tramo no encontrado")
		}

		// 3. Verificar si tiene pasajes activos o emitidos (distintos de ANULADO)
		// Usamos el helper HasActivePasaje que ya ignora los anulados
		if item.HasActivePasaje() {
			return fmt.Errorf("no se puede revertir: tiene pasajes activos o emitidos. Debe anularlos primero.")
		}

		// 4. Revertir aprobación via model
		if !sol.RevertApprovalItem(itemID) {
			return fmt.Errorf("error al revertir aprobación del tramo")
		}

		if err := repoTx.Update(ctx, sol); err != nil {
			return err
		}

		return nil
	})
}

func (s *SolicitudService) RevertFinalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(ctx, id)
		if err != nil {
			return err
		}

		// Revert to EMITIDO via Model
		if err := solicitud.RevertFinalize(); err != nil {
			return err
		}

		if err := tx.Save(solicitud).Error; err != nil {
			return err
		}
		if err := tx.Save(solicitud.Items).Error; err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
func (s *SolicitudService) GetPendingCount(ctx context.Context, userID string, isAdmin bool) (int64, error) {
	return s.repo.CountPending(ctx, userID, isAdmin)
}

func (s *SolicitudService) UpdateSolicitudDates(ctx context.Context, id string, createdAt, updatedAt string) error {
	cA, err := utils.ParseDateTime(createdAt)
	if err != nil {
		return fmt.Errorf("fecha de creación inválida: %w", err)
	}

	uA, err := utils.ParseDateTime(updatedAt)
	if err != nil {
		return fmt.Errorf("fecha de actualización inválida: %w", err)
	}

	return s.repo.UpdateTimestamps(ctx, id, cA, uA)
}

func (s *SolicitudService) UpdateSolicitudItemDates(ctx context.Context, id string, createdAt, updatedAt string) error {
	cA, err := utils.ParseDateTime(createdAt)
	if err != nil {
		return fmt.Errorf("fecha de creación inválida: %w", err)
	}

	uA, err := utils.ParseDateTime(updatedAt)
	if err != nil {
		return fmt.Errorf("fecha de actualización inválida: %w", err)
	}

	return s.solicitudItemRepo.UpdateTimestamps(ctx, id, cA, uA)
}
