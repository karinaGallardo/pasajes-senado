package services

import (
	"context"
	"errors"
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
		repo:              repositories.NewSolicitudRepository(),
		tipoSolicitudRepo: repositories.NewTipoSolicitudRepository(),
		usuarioRepo:       repositories.NewUsuarioRepository(),
		itemRepo:          repositories.NewCupoDerechoItemRepository(),
		tipoItinRepo:      repositories.NewTipoItinerarioRepository(),
	}
}

type SolicitudService struct {
	repo              *repositories.SolicitudRepository
	tipoSolicitudRepo *repositories.TipoSolicitudRepository
	usuarioRepo       *repositories.UsuarioRepository
	itemRepo          *repositories.CupoDerechoItemRepository
	tipoItinRepo      *repositories.TipoItinerarioRepository
}

func (s *SolicitudService) Create(ctx context.Context, req dtos.CreateSolicitudRequest, currentUser *models.Usuario) (*models.Solicitud, error) {
	layout := "2006-01-02T15:04"
	var fechaIda *time.Time
	if t, err := time.Parse(layout, req.FechaIda); err == nil {
		fechaIda = &t
	}

	var fechaVuelta *time.Time
	if req.FechaVuelta != "" {
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
		if itin, _ := s.tipoItinRepo.WithContext(ctx).FindByCodigo("IDA_VUELTA"); itin != nil {
			itinID = itin.ID
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

	var codigo string
	for range 10 {
		generated, err := utils.GenerateCode(5)
		if err != nil {
			return nil, errors.New("error generando código solicitud")
		}
		exists, _ := s.repo.WithContext(ctx).ExistsByCodigo(generated)
		if !exists {
			codigo = generated
			break
		}
	}
	if codigo == "" {
		return nil, errors.New("no se pudo generar un código único después de varios intentos")
	}
	solicitud.Codigo = codigo

	if err := s.repo.WithContext(ctx).Create(solicitud); err != nil {
		return nil, err
	}

	return solicitud, nil
}

func (s *SolicitudService) GetAll(ctx context.Context) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *SolicitudService) GetByUserID(ctx context.Context, userID string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByUserID(userID)
}

func (s *SolicitudService) GetByUserIdOrAccesibleByEncargadoID(ctx context.Context, userID string) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByUserIdOrAccesibleByEncargadoID(userID)
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
	return s.repo.WithContext(ctx).UpdateStatus(id, "RECHAZADO")
}

func (s *SolicitudService) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return s.repo.WithContext(ctx).Update(solicitud)
}

func (s *SolicitudService) Delete(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).Delete(id)
}
