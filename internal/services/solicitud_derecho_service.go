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

type SolicitudDerechoService struct {
	repo                *repositories.SolicitudRepository
	tipoSolicitudRepo   *repositories.TipoSolicitudRepository
	usuarioRepo         *repositories.UsuarioRepository
	itemRepo            *repositories.CupoDerechoItemRepository
	tipoItinRepo        *repositories.TipoItinerarioRepository
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository
	notificationService *NotificationService
	emailService        *EmailService
	baseService         *SolicitudService
	destinoService      *DestinoService
}

func NewSolicitudDerechoService(
	repo *repositories.SolicitudRepository,
	tipoSolicitudRepo *repositories.TipoSolicitudRepository,
	usuarioRepo *repositories.UsuarioRepository,
	itemRepo *repositories.CupoDerechoItemRepository,
	tipoItinRepo *repositories.TipoItinerarioRepository,
	codigoSecuenciaRepo *repositories.CodigoSecuenciaRepository,
	notificationService *NotificationService,
	emailService *EmailService,
	baseService *SolicitudService,
	destinoService *DestinoService,
) *SolicitudDerechoService {
	return &SolicitudDerechoService{
		repo:                repo,
		tipoSolicitudRepo:   tipoSolicitudRepo,
		usuarioRepo:         usuarioRepo,
		itemRepo:            itemRepo,
		tipoItinRepo:        tipoItinRepo,
		codigoSecuenciaRepo: codigoSecuenciaRepo,
		notificationService: notificationService,
		emailService:        emailService,
		baseService:         baseService,
		destinoService:      destinoService,
	}
}

func (s *SolicitudDerechoService) CreateDerecho(ctx context.Context, req dtos.CreateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	req.OrigenIdaIATA = strings.ToUpper(strings.TrimSpace(req.OrigenIdaIATA))
	req.DestinoVueltaIATA = strings.ToUpper(strings.TrimSpace(req.DestinoVueltaIATA))

	fechaIda, err := utils.ParseDateTime(req.FechaIda)
	if err != nil {
		return nil, fmt.Errorf("error en fecha de ida: %w", err)
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" && !req.VueltaPorConfirmar {
		t, err := utils.ParseDateTime(req.FechaVuelta)
		if err != nil {
			return nil, fmt.Errorf("error en fecha de vuelta: %w", err)
		}
		fechaVuelta = t
	}

	realSolicitanteID := req.TargetUserID
	if realSolicitanteID == "" {
		realSolicitanteID = currentUser.ID
	}

	solicitud := &models.Solicitud{
		BaseModel:           models.BaseModel{CreatedBy: &currentUser.ID},
		UsuarioID:           realSolicitanteID,
		TipoSolicitudCodigo: req.TipoSolicitudCodigo,
		AmbitoViajeCodigo:   req.AmbitoViajeCodigo,
		Motivo:              req.Motivo,
		Autorizacion:        req.Autorizacion,
		AerolineaID:         utils.NilIfEmpty(req.AerolineaID),
	}

	if req.CupoDerechoItemID != "" {
		solicitud.CupoDerechoItemID = &req.CupoDerechoItemID
	}

	// Validar beneficiario y permisos

	beneficiary, err := s.usuarioRepo.FindByID(ctx, solicitud.UsuarioID)
	if err != nil {
		return nil, errors.New("usuario beneficiario no encontrado")
	}

	if !beneficiary.IsSenador() {
		return nil, errors.New("solo los Senadores pueden recibir pasajes por derecho")
	}

	if !currentUser.CanCreateSolicitudFor(beneficiary) {
		return nil, errors.New("no tiene permisos para crear una solicitud para este beneficiario")
	}

	solicitud.EstadoSolicitudCodigo = utils.Ptr("SOLICITADO")

	var items []models.SolicitudItem
	hasProgrammed := false

	sede := req.SedeIATA
	if sede == "" {
		return nil, errors.New("es obligatorio definir la sede de la solicitud")
	}

	// Tramo 1: IDA
	stIda := "PENDIENTE"
	if req.FechaIda != "" && !req.IdaPorConfirmar {
		stIda = "SOLICITADO"
		hasProgrammed = true
	}
	items = append(items, models.SolicitudItem{
		Tipo:         models.TipoSolicitudItemIda,
		OrigenIATA:   req.OrigenIdaIATA,
		DestinoIATA:  sede,
		Fecha:        fechaIda,
		EstadoCodigo: utils.Ptr(stIda),
	})

	// Tramo 2: VUELTA
	stVuelta := "PENDIENTE"
	if req.FechaVuelta != "" && !req.VueltaPorConfirmar {
		stVuelta = "SOLICITADO"
		hasProgrammed = true
	}
	items = append(items, models.SolicitudItem{
		Tipo:         models.TipoSolicitudItemVuelta,
		OrigenIATA:   sede,
		DestinoIATA:  req.DestinoVueltaIATA,
		Fecha:        fechaVuelta,
		EstadoCodigo: utils.Ptr(stVuelta),
	})

	if !hasProgrammed {
		return nil, errors.New("debe programar al menos un tramo con fecha definida (Ida o Vuelta)")
	}

	solicitud.Items = items

	err = s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {
		if err := repoTx.CreateWithSequenceCode(ctx, solicitud, "SPD", s.codigoSecuenciaRepo); err != nil {
			return err
		}

		if solicitud.CupoDerechoItemID != nil {
			itemRepoTx := s.itemRepo.WithTx(tx)
			cupoItem, err := itemRepoTx.FindByID(ctx, *solicitud.CupoDerechoItemID)
			if err == nil && cupoItem != nil {
				if cupoItem.IsVencido() && !currentUser.IsAdminOrResponsable() {
					return errors.New("el periodo de este cupo ha vencido. Solo personal administrativo puede registrar solicitudes en periodos anteriores")
				}

				cupoItem.EstadoCupoDerechoCodigo = "RESERVADO"
				if err := itemRepoTx.Update(ctx, cupoItem); err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	go s.notificationService.NotifySolicitudCreated(ctx, solicitud)

	if s.baseService != nil {
		go s.baseService.sendCreationEmail(solicitud)
	}

	return solicitud, nil
}

func (s *SolicitudDerechoService) UpdateDerecho(ctx context.Context, id string, req dtos.UpdateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	solicitud, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("solicitud no encontrada: %w", err)
	}

	if !solicitud.CanEdit(currentUser) {
		return nil, errors.New("no tiene permisos para editar esta solicitud o el estado actual no lo permite")
	}

	// Parsing dates
	var fechaIda *time.Time
	if req.FechaIda != "" && !req.IdaPorConfirmar {
		t, err := utils.ParseDateTime(req.FechaIda)
		if err != nil {
			return nil, errors.New("formato fecha salida inválido")
		}
		fechaIda = t
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" && !req.VueltaPorConfirmar {
		t, err := utils.ParseDateTime(req.FechaVuelta)
		if err != nil {
			return nil, errors.New("formato fecha retorno inválido")
		}
		fechaVuelta = t
		if fechaVuelta != nil && fechaIda != nil && !fechaVuelta.After(*fechaIda) {
			return nil, errors.New("la fecha de retorno debe ser posterior a la de salida")
		}
	}

	// Business Logic: Solo actualizamos campos base si el estado lo permite
	if solicitud.IsSolicitado() || solicitud.IsAprobado() || solicitud.IsParcialmenteAprobado() || solicitud.IsRechazado() {
		// If it was already approved/rejected, we revert to SOLICITADO to trigger a new approval flow because it was edited.
		if solicitud.IsAprobado() || solicitud.IsParcialmenteAprobado() || solicitud.IsRechazado() {
			solicitud.EstadoSolicitudCodigo = utils.Ptr("SOLICITADO")
		}

		solicitud.TipoSolicitudCodigo = req.TipoSolicitudCodigo
		solicitud.AmbitoViajeCodigo = req.AmbitoViajeCodigo
		solicitud.AerolineaID = utils.NilIfEmpty(req.AerolineaID)
		if req.Motivo != "" {
			solicitud.Motivo = req.Motivo
		}

		req.OrigenIdaIATA = strings.ToUpper(strings.TrimSpace(req.OrigenIdaIATA))
		req.DestinoVueltaIATA = strings.ToUpper(strings.TrimSpace(req.DestinoVueltaIATA))
		sede := req.SedeIATA
		if sede == "" {
			return nil, errors.New("la sede de la solicitud no puede estar vacía")
		}

		// Rule: DERECHO always needs IDA and VUELTA placeholders if missing
		s.ensureDerechoItems(solicitud, req.OrigenIdaIATA, req.DestinoVueltaIATA, sede)

		hasProgrammed := false
		// Synchronize items
		for i := range solicitud.Items {
			it := &solicitud.Items[i]
			if !it.CanEdit() {
				// Si un tramo ya no es editable (ej: emitido), cuenta como programado
				hasProgrammed = true
				continue
			}

			switch it.Tipo {
			case models.TipoSolicitudItemIda:
				it.OrigenIATA = req.OrigenIdaIATA
				it.DestinoIATA = sede
				it.Origen = nil
				it.Destino = nil

				if req.IdaPorConfirmar {
					it.EstadoCodigo = utils.Ptr("PENDIENTE")
					it.Fecha = nil
				} else {
					it.EstadoCodigo = utils.Ptr("SOLICITADO")
					it.Fecha = fechaIda
				}
			case models.TipoSolicitudItemVuelta:
				it.OrigenIATA = sede
				it.DestinoIATA = req.DestinoVueltaIATA
				it.Origen = nil
				it.Destino = nil

				if req.VueltaPorConfirmar {
					it.EstadoCodigo = utils.Ptr("PENDIENTE")
					it.Fecha = nil
				} else {
					it.EstadoCodigo = utils.Ptr("SOLICITADO")
					it.Fecha = fechaVuelta
				}
			}

			if it.Fecha != nil {
				hasProgrammed = true
			}
		}

		if !hasProgrammed {
			return nil, errors.New("debe programar al menos un tramo con fecha definida (Ida o Vuelta)")
		}
	}

	// El Hook BeforeUpdate en el modelo se encargará de recalcular TipoItinerarioCodigo y EstadoSolicitudCodigo
	if err := s.repo.Update(ctx, solicitud); err != nil {
		return nil, fmt.Errorf("error al actualizar solicitud: %w", err)
	}

	return solicitud, nil
}

func (s *SolicitudDerechoService) ensureDerechoItems(solicitud *models.Solicitud, origenIda, destinoVuelta, sede string) {
	if solicitud.GetItemIda() == nil {
		solicitud.Items = append(solicitud.Items, models.SolicitudItem{
			SolicitudID:  solicitud.ID,
			Tipo:         models.TipoSolicitudItemIda,
			OrigenIATA:   origenIda,
			DestinoIATA:  sede,
			EstadoCodigo: utils.Ptr("PENDIENTE"),
		})
	}
	if solicitud.GetItemVuelta() == nil {
		solicitud.Items = append(solicitud.Items, models.SolicitudItem{
			SolicitudID:  solicitud.ID,
			Tipo:         models.TipoSolicitudItemVuelta,
			OrigenIATA:   sede,
			DestinoIATA:  destinoVuelta,
			EstadoCodigo: utils.Ptr("PENDIENTE"),
		})
	}
}
