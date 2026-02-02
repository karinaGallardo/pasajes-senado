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
	emailService        *EmailService
}

func (s *SolicitudService) Create(ctx context.Context, req dtos.CreateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" && req.TipoItinerarioCode != "SOLO_IDA" {
		if t, err := time.Parse(layout, req.FechaVuelta); err == nil {
			fechaVuelta = &t
		}
	}

	realSolicitanteID := req.TargetUserID
	if realSolicitanteID == "" {
		realSolicitanteID = currentUser.ID
	}

	itinID := req.TipoItinerarioID
	if itinID == "" {
		if req.TipoItinerarioCode != "" {
			if itin, _ := s.tipoItinRepo.WithContext(ctx).FindByCodigo(req.TipoItinerarioCode); itin != nil {
				itinID = itin.ID
			}
		}
		// Fallback
		if itinID == "" {
			if itin, _ := s.tipoItinRepo.WithContext(ctx).FindByCodigo("IDA_VUELTA"); itin != nil {
				itinID = itin.ID
			}
		}
	}

	solicitud := &models.Solicitud{
		BaseModel:         models.BaseModel{CreatedBy: &currentUser.ID},
		UsuarioID:         realSolicitanteID,
		TipoSolicitudID:   req.TipoSolicitudID,
		AmbitoViajeID:     req.AmbitoViajeID,
		TipoItinerarioID:  itinID,
		OrigenIATA:        req.OrigenIATA,
		DestinoIATA:       req.DestinoIATA,
		FechaIda:          fechaIda,
		FechaVuelta:       fechaVuelta,
		Motivo:            req.Motivo,
		AerolineaSugerida: req.AerolineaSugerida,
		Autorizacion:      req.Autorizacion,
	}

	if req.CupoDerechoItemID != "" {
		solicitud.CupoDerechoItemID = &req.CupoDerechoItemID
	}

	tipoSolicitud, err := s.tipoSolicitudRepo.WithContext(ctx).FindByID(solicitud.TipoSolicitudID)
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

	rut := fmt.Sprintf("%s -> %s", fullSol.Origen.Ciudad, fullSol.Destino.Ciudad)
	fecha := "-"
	if fullSol.FechaIda != nil {
		fecha = utils.FormatDateShortES(*fullSol.FechaIda)
	}

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; color: #333;">
			<h2>Solicitud de Pasaje Registrada</h2>
			<p>Hola <strong>%s</strong>,</p>
			<p>Se ha registrado una nueva solicitud de pasaje a su nombre con los siguientes detalles:</p>
			<table style="border-collapse: collapse; width: 100%%; max-width: 600px;">
				<tr>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Código:</strong></td>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;">%s</td>
				</tr>
				<tr>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Ruta:</strong></td>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;">%s</td>
				</tr>
				<tr>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Fecha Ida:</strong></td>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;">%s</td>
				</tr>
				<tr>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;"><strong>Estado:</strong></td>
					<td style="padding: 8px; border-bottom: 1px solid #ddd;">%s</td>
				</tr>
			</table>
			<p style="margin-top: 20px;">Puede revisar el estado de su solicitud ingresando al sistema.</p>
		</div>
	`, beneficiary.GetNombreCompleto(), fullSol.Codigo, rut, fecha, fullSol.GetEstado())

	err = s.emailService.SendEmail([]string{beneficiary.Email}, nil, nil, subject, body)
	if err != nil {
		fmt.Printf("Error sending email: %v\n", err)
	}
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

func (s *SolicitudService) Approve(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		itemRepoTx := s.itemRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		if err := repoTx.UpdateStatus(id, "APROBADO"); err != nil {
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
	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		if *solicitud.EstadoSolicitudCodigo != "APROBADO" {
			return errors.New("la solicitud no está en estado APROBADO")
		}

		if len(solicitud.Pasajes) > 0 {
			return errors.New("no se puede revertir la aprobación porque ya tiene pasajes asignados")
		}

		if err := repoTx.UpdateStatus(id, "SOLICITADO"); err != nil {
			return err
		}

		return nil
	})
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
	solicitud, err := s.repo.WithContext(ctx).FindByID(id)
	if err != nil {
		return err
	}

	if err := s.repo.WithContext(ctx).UpdateStatus(id, "RECHAZADO"); err != nil {
		return err
	}

	if solicitud.CupoDerechoItemID != nil {
		item, err := s.itemRepo.WithContext(ctx).FindByID(*solicitud.CupoDerechoItemID)
		if err == nil && item != nil {
			item.EstadoCupoDerechoCodigo = "DISPONIBLE"
			_ = s.itemRepo.WithContext(ctx).Update(item)
		}
	}
	return nil
}

func (s *SolicitudService) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return s.repo.WithContext(ctx).Update(solicitud)
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
