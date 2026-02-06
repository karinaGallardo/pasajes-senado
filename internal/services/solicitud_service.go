package services

import (
	"context"
	"errors"
	"fmt"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
	"time"

	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func NewSolicitudService() *SolicitudService {
	return &SolicitudService{
		repo:                repositories.NewSolicitudRepository(),
		tipoSolicitudRepo:   repositories.NewTipoSolicitudRepository(),
		usuarioRepo:         repositories.NewUsuarioRepository(),
		itemRepo:            repositories.NewCupoDerechoItemRepository(),
		tipoItinRepo:        repositories.NewTipoItinerarioRepository(),
		codigoSecuenciaRepo: repositories.NewCodigoSecuenciaRepository(),
		solicitudItemRepo:   repositories.NewSolicitudItemRepository(),
		pasajeRepo:          repositories.NewPasajeRepository(),
		emailService:        NewEmailService(),
	}
}

type SolicitudService struct {
	repo                *repositories.SolicitudRepository
	tipoSolicitudRepo   *repositories.TipoSolicitudRepository
	usuarioRepo         *repositories.UsuarioRepository
	itemRepo            *repositories.CupoDerechoItemRepository
	tipoItinRepo        *repositories.TipoItinerarioRepository
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository
	solicitudItemRepo   *repositories.SolicitudItemRepository
	pasajeRepo          *repositories.PasajeRepository
	emailService        *EmailService
}

func (s *SolicitudService) Create(ctx context.Context, req dtos.CreateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" && req.TipoItinerarioCodigo != "SOLO_IDA" {
		if t, err := time.Parse(layout, req.FechaVuelta); err == nil {
			fechaVuelta = &t
		}
	}

	realSolicitanteID := req.TargetUserID
	if realSolicitanteID == "" {
		realSolicitanteID = currentUser.ID
	}

	// Resolve TipoItinerario
	var tipoItin *models.TipoItinerario
	if req.TipoItinerarioCodigo != "" {
		tipoItin, _ = s.tipoItinRepo.WithContext(ctx).FindByCodigo(req.TipoItinerarioCodigo)
	}
	if tipoItin == nil && req.TipoItinerario != "" {
		tipoItin, _ = s.tipoItinRepo.WithContext(ctx).FindByCodigo(req.TipoItinerario)
	}
	if tipoItin == nil {
		tipoItin, _ = s.tipoItinRepo.WithContext(ctx).FindByCodigo("IDA_VUELTA")
	}

	if tipoItin == nil {
		return nil, errors.New("tipo de itinerario no válido")
	}

	solicitud := &models.Solicitud{
		BaseModel:            models.BaseModel{CreatedBy: &currentUser.ID},
		UsuarioID:            realSolicitanteID,
		TipoSolicitudCodigo:  req.TipoSolicitudCodigo,
		AmbitoViajeCodigo:    req.AmbitoViajeCodigo,
		TipoItinerarioCodigo: tipoItin.Codigo,
		Motivo:               req.Motivo,
		Autorizacion:         req.Autorizacion,
		AerolineaSugerida:    req.AerolineaSugerida,
	}

	if req.CupoDerechoItemID != "" {
		solicitud.CupoDerechoItemID = &req.CupoDerechoItemID
	}

	tipoSolicitud, err := s.tipoSolicitudRepo.WithContext(ctx).FindByID(solicitud.TipoSolicitudCodigo)
	if err != nil {
		return nil, errors.New("tipo de solicitud inválido o no encontrado")
	}

	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		beneficiary, err := s.usuarioRepo.WithContext(ctx).FindByID(solicitud.UsuarioID)
		if err != nil {
			return nil, errors.New("usuario beneficiario no encontrado")
		}

		if !strings.Contains(beneficiary.Tipo, "SENADOR") {
			return nil, errors.New("solo los Senadores pueden recibir pasajes por derecho")
		}

		canCreate := false
		if currentUser.ID == beneficiary.ID ||
			currentUser.IsAdminOrResponsable() ||
			(beneficiary.EncargadoID != nil && *beneficiary.EncargadoID == currentUser.ID) {
			canCreate = true
		}

		if !canCreate {
			return nil, errors.New("no tiene autorización para emitir solicitudes de pasajes para este Senador")
		}
	}

	solicitud.EstadoSolicitudCodigo = utils.Ptr("SOLICITADO")

	// Build Items
	var items []models.SolicitudItem
	itinCode := tipoItin.Codigo

	// Common: Tramo 1 (IDA or VUELTA single)
	switch itinCode {
	case "SOLO_IDA", "IDA_VUELTA":
		items = append(items, models.SolicitudItem{
			Tipo:         models.TipoSolicitudItemIda,
			OrigenIATA:   req.OrigenIATA,
			DestinoIATA:  req.DestinoIATA,
			Fecha:        fechaIda,
			Hora:         s.formatTime(fechaIda),
			EstadoCodigo: utils.Ptr("SOLICITADO"),
		})
	case "SOLO_VUELTA":
		// Assumption: User input for Origin/Dest matches the leg (e.g. Origin=MIA, Dest=VVI for Return)
		items = append(items, models.SolicitudItem{
			Tipo:         models.TipoSolicitudItemVuelta,
			OrigenIATA:   req.OrigenIATA,
			DestinoIATA:  req.DestinoIATA,
			Fecha:        fechaIda, // Single date input usually
			Hora:         s.formatTime(fechaIda),
			EstadoCodigo: utils.Ptr("SOLICITADO"),
		})
	}

	// Tramo 2: VUELTA (if Round Trip)
	if itinCode == "IDA_VUELTA" {
		st := "SOLICITADO"
		if fechaVuelta == nil {
			st = "PENDIENTE"
		}
		items = append(items, models.SolicitudItem{
			Tipo:         models.TipoSolicitudItemVuelta,
			OrigenIATA:   req.DestinoIATA, // Swap
			DestinoIATA:  req.OrigenIATA,  // Swap
			Fecha:        fechaVuelta,
			Hora:         s.formatTime(fechaVuelta),
			EstadoCodigo: utils.Ptr(st),
		})
	}

	solicitud.Items = items

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		secuenciaRepoTx := s.codigoSecuenciaRepo.WithTx(tx)
		itemRepoTx := s.itemRepo.WithTx(tx)

		currentYear := time.Now().Year()
		nextVal, err := secuenciaRepoTx.GetNext(currentYear, "SPD")
		if err != nil {
			return errors.New("error generando codigo de secuencia de solicitud")
		}

		solicitud.Codigo = fmt.Sprintf("SPD-%d%04d", currentYear%100, nextVal)

		if err := repoTx.Create(solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(item); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	go s.sendCreationEmail(solicitud)

	return solicitud, nil
}

// Helper locally if not in utils
func (SolicitudService) formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("15:04")
}

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

	subject := fmt.Sprintf("Solicitud de Pasaje Creada - %s", fullSol.Codigo)

	rut := "-"
	fecha := "-"
	var mainItem *models.SolicitudItem

	for i := range fullSol.Items {
		it := &fullSol.Items[i]
		if it.Tipo == models.TipoSolicitudItemIda {
			mainItem = it
			break
		}
	}
	if mainItem == nil && len(fullSol.Items) > 0 {
		mainItem = &fullSol.Items[0]
	}

	if mainItem != nil {
		origen := "-"
		if mainItem.Origen != nil {
			origen = mainItem.Origen.Ciudad
		}
		destino := "-"
		if mainItem.Destino != nil {
			destino = mainItem.Destino.Ciudad
		}

		rut = fmt.Sprintf("%s -> %s", origen, destino)

		if mainItem.Fecha != nil {
			fecha = utils.FormatDateShortES(*mainItem.Fecha)
		}
	}

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solURL := fmt.Sprintf("%s/solicitudes/derecho/%s/detalle", baseURL, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #03738C; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Solicitud de Pasaje Registrada</h2>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>Se ha registrado una nueva solicitud de pasaje a su nombre con los siguientes detalles:</p>
				<table style="border-collapse: collapse; width: 100%%; margin-bottom: 20px;">
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Código:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Ruta:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Fecha Ida:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
					<tr>
						<td style="padding: 8px; border-bottom: 1px solid #eee;"><strong>Estado:</strong></td>
						<td style="padding: 8px; border-bottom: 1px solid #eee;">%s</td>
					</tr>
				</table>

				<div style="margin-top: 25px; text-align: center;">
					<a href="%s" target="_blank" style="background-color: #03738C; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">Ver Detalles en el Sistema</a>
				</div>
				<p style="font-size: 11px; color: #999; margin-top: 20px; text-align: center;">
					Si el botón no funciona, copie y pegue esta URL:<br>
					%s
				</p>
			</div>
		</div>
	`, beneficiary.GetNombreCompleto(), fullSol.Codigo, rut, fecha, fullSol.GetEstado(), solURL, solURL)

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

	subject := fmt.Sprintf("Solicitud de Pasaje Revertida - %s", fullSol.Codigo)

	baseURL := viper.GetString("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8284"
	}
	solURL := fmt.Sprintf("%s/solicitudes/derecho/%s/detalle", baseURL, fullSol.ID)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333; max-width: 600px;">
			<div style="background-color: #d97706; color: white; padding: 15px; border-radius: 5px 5px 0 0;">
				<h2 style="margin:0;">Aprobación Revertida</h2>
			</div>
			<div style="padding: 20px; border: 1px solid #ddd; border-top: none; border-radius: 0 0 5px 5px;">
				<p>Hola <strong>%s</strong>,</p>
				<p>La aprobación de su solicitud <strong>%s</strong> ha sido revertida al estado <strong>SOLICITADO</strong>.</p>
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
	`, beneficiary.GetNombreCompleto(), fullSol.Codigo, solURL, solURL)

	_ = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
}

func (s *SolicitudService) GetAll(ctx context.Context, status string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindAll(status)
}

func (s *SolicitudService) GetByUserID(ctx context.Context, userID string, status string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByUserID(userID, status)
}

func (s *SolicitudService) GetByUserIdOrAccesibleByEncargadoID(ctx context.Context, userID string, status string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByUserIdOrAccesibleByEncargadoID(userID, status)
}

func (s *SolicitudService) GetByCupoDerechoItemID(ctx context.Context, itemID string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByCupoDerechoItemID(itemID)
}

func (s *SolicitudService) GetByID(ctx context.Context, id string) (*models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *SolicitudService) GetItemByID(ctx context.Context, id string) (*models.SolicitudItem, error) {
	return s.solicitudItemRepo.FindByID(id)
}

func (s *SolicitudService) Approve(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		hasIda := false
		hasVuelta := false
		for i := range solicitud.Items {
			switch solicitud.Items[i].Tipo {
			case models.TipoSolicitudItemIda:
				hasIda = true
			case models.TipoSolicitudItemVuelta:
				hasVuelta = true
			}
		}

		approveAll := hasIda && hasVuelta

		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			shouldApprove := false

			if approveAll {
				shouldApprove = true
			} else {
				// Approve only if it has a confirmed date (i.e., not PENDING placeholder)
				// Re-using the PENDIENTE status logic we added
				if item.GetEstado() != "PENDIENTE" {
					shouldApprove = true
				}
			}

			if shouldApprove && item.GetEstado() == "SOLICITADO" {
				st := "APROBADO"
				item.EstadoCodigo = &st
				if err := tx.Save(item).Error; err != nil {
					return err
				}
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) RevertApproval(ctx context.Context, id string) error {
	var solicitudForEmail *models.Solicitud
	err := s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		st := ""
		if solicitud.EstadoSolicitudCodigo != nil {
			st = *solicitud.EstadoSolicitudCodigo
		}

		if st != "APROBADO" && st != "PARCIALMENTE_APROBADO" && st != "EMITIDO" {
			return errors.New("la solicitud no está en un estado que permita revertir la aprobación (" + st + ")")
		}

		hasPasajes := false
		for _, t := range solicitud.Items {
			if len(t.Pasajes) > 0 {
				hasPasajes = true
				break
			}
		}

		if hasPasajes {
			return errors.New("no se puede revertir la aprobación porque ya tiene pasajes asignados")
		}

		// Revert items
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			if item.GetEstado() == "APROBADO" {
				st := "SOLICITADO"
				item.EstadoCodigo = &st
				if err := tx.Save(item).Error; err != nil {
					return err
				}
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}
		solicitudForEmail = solicitud
		return nil
	})

	if err == nil && solicitudForEmail != nil {
		go s.sendRevertApprovalEmail(solicitudForEmail)
	}

	return err
}

func (s *SolicitudService) Finalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		if err := repoTx.UpdateStatus(id, "FINALIZADO"); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "USADO"
				if err := itemRepoTx.Update(item); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) Reject(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		// Reject all items
		for i := range solicitud.Items {
			item := &solicitud.Items[i]
			st := "RECHAZADO"
			item.EstadoCodigo = &st
			if err := tx.Save(item).Error; err != nil {
				return err
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(item); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (s *SolicitudService) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// Update parent
		if err := tx.Save(solicitud).Error; err != nil {
			return err
		}

		// Update items explicitly
		for i := range solicitud.Items {
			if err := tx.Save(&solicitud.Items[i]).Error; err != nil {
				return err
			}
		}

		solicitud.UpdateStatusBasedOnItems()
		return tx.Save(solicitud).Error
	})
}

func (s *SolicitudService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err == nil && solicitud != nil && solicitud.CupoDerechoItemID != nil {
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				if err := itemRepoTx.Update(item); err != nil {
					return err
				}
			}
		}

		return repoTx.Delete(id)
	})
}

func (s *SolicitudService) ApproveItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(solicitudID)
		if err != nil {
			return err
		}
		found := false
		for i := range solicitud.Items {
			if solicitud.Items[i].ID == itemID {
				st := "APROBADO"
				solicitud.Items[i].EstadoCodigo = &st
				if err := tx.Save(&solicitud.Items[i]).Error; err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item no encontrado")
		}
		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}

		// Si es derecho, marcar como reservado
		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "RESERVADO"
				itemRepoTx.Update(item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RejectItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(solicitudID)
		if err != nil {
			return err
		}
		found := false
		for i := range solicitud.Items {
			if solicitud.Items[i].ID == itemID {
				st := "RECHAZADO"
				solicitud.Items[i].EstadoCodigo = &st
				if err := tx.Save(&solicitud.Items[i]).Error; err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item no encontrado")
		}
		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}
		// Si todos los items activos son rechazados, liberar el cupo
		allRejected := true
		for _, it := range solicitud.Items {
			st := it.GetEstado()
			if st != "RECHAZADO" && st != "CANCELADO" && st != "PENDIENTE" {
				allRejected = false
				break
			}
		}
		if allRejected && solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			item, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && item != nil {
				item.EstadoCupoDerechoCodigo = "DISPONIBLE"
				itemRepoTx.Update(item)
			}
		}
		return nil
	})
}

func (s *SolicitudService) RevertApprovalItem(ctx context.Context, solicitudID, itemID string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(solicitudID)
		if err != nil {
			return err
		}
		found := false
		for i := range solicitud.Items {
			it := &solicitud.Items[i]
			if it.ID == itemID {
				estado := it.GetEstado()
				if estado != "APROBADO" && estado != "EMITIDO" {
					return fmt.Errorf("el tramo no está en un estado que permita revertir la aprobación (%s)", estado)
				}
				if len(it.Pasajes) > 0 {
					return fmt.Errorf("no se puede revertir la aprobación porque ya tiene pasajes asignados")
				}

				st := "SOLICITADO"
				it.EstadoCodigo = &st
				if err := tx.Save(it).Error; err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("item no encontrado")
		}

		solicitud.UpdateStatusBasedOnItems()
		if err := repoTx.Update(solicitud); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			cupoItem, err := itemRepoTx.FindByID(*solicitud.CupoDerechoItemID)
			if err == nil && cupoItem != nil {
				if solicitud.GetEstado() == "SOLICITADO" {
					cupoItem.EstadoCupoDerechoCodigo = "DISPONIBLE"
					itemRepoTx.Update(cupoItem)
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) ReprogramarItem(ctx context.Context, req dtos.ReprogramarSolicitudItemRequest) error {
	stateReprog := "REPROGRAMADO"
	pasajeAnulado := "ANULADO"
	fecha := utils.ParseDate("2006-01-02", req.Fecha)

	return s.repo.RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		// 1. Find old Item
		var oldItem models.SolicitudItem
		if err := tx.First(&oldItem, "id = ?", req.SolicitudItemID).Error; err != nil {
			return fmt.Errorf("item de solicitud no encontrado: %v", err)
		}

		// 2. Mark old item as REPROGRAMADO
		if err := tx.Model(&oldItem).Update("estado_codigo", stateReprog).Error; err != nil {
			return fmt.Errorf("error marcando item anterior como reprogramado: %v", err)
		}

		// 3. Find and annul any associated Pasajes (issued tickets)
		var pasajes []models.Pasaje
		if err := tx.Find(&pasajes, "solicitud_item_id = ? AND estado_pasaje_codigo = 'EMITIDO'", oldItem.ID).Error; err == nil {
			for i := range pasajes {
				p := &pasajes[i]
				p.EstadoPasajeCodigo = &pasajeAnulado
				if req.Motivo != "" {
					p.Glosa += " | Reprogramado. Motivo: " + req.Motivo
				}
				if err := tx.Save(p).Error; err != nil {
					return fmt.Errorf("error anulando pasaje: %v", err)
				}
			}
		}

		// 4. Create NEW SolicitudItem
		newItem := models.SolicitudItem{
			SolicitudID:  oldItem.SolicitudID,
			Tipo:         oldItem.Tipo,
			OrigenIATA:   oldItem.OrigenIATA,
			DestinoIATA:  oldItem.DestinoIATA,
			Fecha:        &fecha,
			Hora:         req.Hora,
			EstadoCodigo: utils.Ptr("SOLICITADO"),
		}

		if err := tx.Create(&newItem).Error; err != nil {
			return fmt.Errorf("error creando nuevo item de solicitud: %v", err)
		}

		return nil
	})
}
