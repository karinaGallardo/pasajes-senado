package services

import (
	"context"
	"encoding/json"
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
	openTicketRepo      *repositories.OpenTicketRepository
	configService       *ConfiguracionService
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
	openTicketRepo *repositories.OpenTicketRepository,
	configService *ConfiguracionService,
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
		openTicketRepo:      openTicketRepo,
		configService:       configService,
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

	// Tramo 1: IDA (opcional si es solo vuelta)
	if !req.SoloVuelta {
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
			AerolineaID:  utils.NilIfEmpty(req.IdaAerolineaID),
			OpenTicketID: utils.NilIfEmpty(req.IdaOpenTicketID),
		})
	}

	// Tramo 2: VUELTA (opcional si es solo ida)
	if !req.SoloIda {
		origenVuelta := sede
		destinoVuelta := req.DestinoVueltaIATA
		if destinoVuelta == "" {
			destinoVuelta = req.OrigenIdaIATA // si no se especifica, vuelve al origen
		}
		stVuelta := "PENDIENTE"
		if req.FechaVuelta != "" && !req.VueltaPorConfirmar {
			stVuelta = "SOLICITADO"
			hasProgrammed = true
		}
		items = append(items, models.SolicitudItem{
			Tipo:         models.TipoSolicitudItemVuelta,
			OrigenIATA:   origenVuelta,
			DestinoIATA:  destinoVuelta,
			Fecha:        fechaVuelta,
			EstadoCodigo: utils.Ptr(stVuelta),
			AerolineaID:  utils.NilIfEmpty(req.VueltaAerolineaID),
			OpenTicketID: utils.NilIfEmpty(req.VueltaOpenTicketID),
		})
	}

	// Tramos extra con crédito OT
	if req.TramosExtraJSON != "" {
		var extras []dtos.TramoExtraRequest
		if err := json.Unmarshal([]byte(req.TramosExtraJSON), &extras); err == nil {
			for _, ex := range extras {
				var fecha *time.Time
				if ex.FechaSalida != "" && !ex.PorConfirmar {
					t, err := utils.ParseDateTime(ex.FechaSalida)
					if err == nil {
						fecha = t
					}
				}
				st := "PENDIENTE"
				if fecha != nil {
					st = "SOLICITADO"
					hasProgrammed = true
				}
				items = append(items, models.SolicitudItem{
					Tipo:         "EXTRA",
					OrigenIATA:   strings.ToUpper(strings.TrimSpace(ex.OrigenIATA)),
					DestinoIATA:  strings.ToUpper(strings.TrimSpace(ex.DestinoIATA)),
					Fecha:        fecha,
					EstadoCodigo: utils.Ptr(st),
					AerolineaID:  utils.NilIfEmpty(ex.AerolineaID),
					OpenTicketID: utils.NilIfEmpty(ex.OpenTicketID),
				})
			}
		}
	}

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

	// Reservar OTs asignados a los tramos
	for _, item := range solicitud.Items {
		if item.OpenTicketID != nil && *item.OpenTicketID != "" {
			s.reserveOpenTicket(ctx, *item.OpenTicketID, solicitud.ID, currentUser.ID)
		}
	}

	go s.notificationService.NotifySolicitudCreated(ctx, solicitud)

	if s.baseService != nil {
		go s.baseService.sendCreationEmail(solicitud)
	}

	return solicitud, nil
}

func (s *SolicitudDerechoService) reserveOpenTicket(ctx context.Context, otID, solicitudID, userID string) {
	ot, err := s.openTicketRepo.FindByID(ctx, otID)
	if err != nil {
		return
	}
	if ot.Estado == models.EstadoOpenTicketDisponible {
		ot.Estado = models.EstadoOpenTicketReservado
		ot.SolicitudConsumoID = &solicitudID
		ot.UpdatedBy = &userID
		_ = s.openTicketRepo.Update(ctx, ot)
	}
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
					it.CreatedAt = time.Now()
					it.Fecha = fechaIda
					hasProgrammed = true
				}
				it.AerolineaID = utils.NilIfEmpty(req.IdaAerolineaID)
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
					it.CreatedAt = time.Now()
					it.Fecha = fechaVuelta
					hasProgrammed = true
				}
				it.AerolineaID = utils.NilIfEmpty(req.VueltaAerolineaID)
			}
		}

		if !hasProgrammed {
			return nil, errors.New("debe programar al menos un tramo con fecha definida (Ida o Vuelta)")
		}
	}

	// El Hook BeforeUpdate en el modelo se encargará de recalcular TipoItinerarioCodigo y EstadoSolicitudCodigo
	err = s.repo.RunTransaction(func(repoTx *repositories.SolicitudRepository, tx *gorm.DB) error {

		if err := repoTx.Update(ctx, solicitud); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
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
			AerolineaID:  solicitud.AerolineaID,
		})
	}
	if solicitud.GetItemVuelta() == nil {
		solicitud.Items = append(solicitud.Items, models.SolicitudItem{
			SolicitudID:  solicitud.ID,
			Tipo:         models.TipoSolicitudItemVuelta,
			OrigenIATA:   sede,
			DestinoIATA:  destinoVuelta,
			EstadoCodigo: utils.Ptr("PENDIENTE"),
			AerolineaID:  solicitud.AerolineaID,
		})
	}
}

func (s *SolicitudDerechoService) GetDefaultTravelDates() (string, string) {
	ida := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	vuelta := time.Now().AddDate(0, 0, 4).Format("2006-01-02")
	return ida, vuelta
}

func (s *SolicitudDerechoService) GetSedesAutorizadas(ctx context.Context) []models.Destino {
	sedesAutorizadas := []models.Destino{}
	var sedesCodes []string

	sedesStr := ""
	if s.configService != nil {
		sedesStr = s.configService.GetValue(ctx, "SEDES_AUTORIZADAS")
	}

	if sedesStr == "" {
		sedesCodes = []string{"LPB"}
	} else {
		for _, p := range strings.Split(sedesStr, ",") {
			code := strings.ToUpper(strings.TrimSpace(p))
			if code != "" {
				sedesCodes = append(sedesCodes, code)
			}
		}
	}

	for _, code := range sedesCodes {
		if dest, err := s.destinoService.GetByIATA(ctx, code); err == nil && dest != nil {
			sedesAutorizadas = append(sedesAutorizadas, *dest)
		}
	}
	return sedesAutorizadas
}

type EditFormDefaults struct {
	Origen  *models.Destino
	Destino *models.Destino
	Sede    *models.Destino
}

func (s *SolicitudDerechoService) GetEditFormDefaults(ctx context.Context, solicitud *models.Solicitud, userLoc *models.Destino) *EditFormDefaults {
	result := &EditFormDefaults{}
	ida := solicitud.GetItemIda()
	vuelta := solicitud.GetItemVuelta()
	sedesAutorizadas := s.GetSedesAutorizadas(ctx)

	var sedeDefault *models.Destino
	if len(sedesAutorizadas) > 0 {
		sedeDefault = &sedesAutorizadas[0]
	}

	if ida != nil {
		result.Origen = ida.Origen
		result.Sede = ida.Destino
	}

	if vuelta != nil {
		if result.Origen == nil {
			result.Origen = vuelta.Destino
		}
		result.Destino = vuelta.Destino
		if result.Sede == nil {
			result.Sede = vuelta.Origen
		}
	}

	if result.Origen == nil {
		result.Origen = userLoc
	}

	if result.Destino == nil {
		if vuelta != nil && vuelta.Destino != nil {
			result.Destino = vuelta.Destino
		} else if ida != nil && ida.Destino != nil {
			result.Destino = ida.Destino
		} else {
			result.Destino = sedeDefault
		}
	}

	if result.Sede == nil {
		result.Sede = sedeDefault
	}

	return result
}
