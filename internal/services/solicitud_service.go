package services

import (
	"context"
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"

	"gorm.io/gorm"
)

type SolicitudService struct {
	repo              *repositories.SolicitudRepository
	tipoSolicitudRepo *repositories.TipoSolicitudRepository
	usuarioRepo       *repositories.UsuarioRepository
	voucherRepo       *repositories.AsignacionVoucherRepository
}

func NewSolicitudService() *SolicitudService {
	return &SolicitudService{
		repo:              repositories.NewSolicitudRepository(),
		tipoSolicitudRepo: repositories.NewTipoSolicitudRepository(),
		usuarioRepo:       repositories.NewUsuarioRepository(),
		voucherRepo:       repositories.NewAsignacionVoucherRepository(),
	}
}

func (s *SolicitudService) Create(ctx context.Context, solicitud *models.Solicitud, usuario *models.Usuario, voucherID string) error {
	tipoSolicitud, err := s.tipoSolicitudRepo.WithContext(ctx).FindByID(solicitud.TipoSolicitudID)
	if err != nil {
		return errors.New("tipo de solicitud inválido o no encontrado")
	}

	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		beneficiary, err := s.usuarioRepo.WithContext(ctx).FindByID(solicitud.UsuarioID)
		if err != nil {
			return errors.New("usuario beneficiario no encontrado")
		}

		esSenador := strings.Contains(beneficiary.Tipo, "SENADOR")
		if !esSenador {
			return errors.New("solo los Senadores pueden recibir pasajes por derecho")
		}

		canCreate := false
		if usuario.ID == beneficiary.ID {
			canCreate = true
		} else if usuario.RolCodigo != nil && (*usuario.RolCodigo == "ADMIN" || *usuario.RolCodigo == "TECNICO") {
			canCreate = true
		} else if beneficiary.EncargadoID != nil && *beneficiary.EncargadoID == usuario.ID {
			canCreate = true
		}

		if !canCreate {
			return errors.New("no tiene autorización para emitir solicitudes de pasajes para este Senador")
		}
	}

	estadoSolicitado := "SOLICITADO"
	solicitud.EstadoSolicitudCodigo = &estadoSolicitado

	var codigo string
	for range 10 {
		generated, err := utils.GenerateCode(5)
		if err != nil {
			return errors.New("error generando código solicitud")
		}
		exists, _ := s.repo.WithContext(ctx).ExistsByCodigo(generated)
		if !exists {
			codigo = generated
			break
		}
	}
	if codigo == "" {
		return errors.New("no se pudo generar un código único después de varios intentos")
	}
	solicitud.Codigo = codigo

	if voucherID != "" {
		solicitud.VoucherID = &voucherID
	}

	if err := s.repo.WithContext(ctx).Create(solicitud); err != nil {
		return err
	}

	return nil
}

func (s *SolicitudService) FindAll(ctx context.Context) ([]models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindAll()
}

func (s *SolicitudService) FindByID(ctx context.Context, id string) (*models.Solicitud, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *SolicitudService) Approve(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).GetDB().Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)
		voucherRepoTx := s.voucherRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		estadoAprobado := "APROBADO"
		if err := repoTx.UpdateStatus(id, estadoAprobado); err != nil {
			return err
		}

		if solicitud.VoucherID != nil {
			voucher, err := voucherRepoTx.FindByID(*solicitud.VoucherID)
			if err == nil && voucher != nil {
				estadoReservado := "RESERVADO"
				voucher.EstadoVoucherCodigo = estadoReservado
				if err := voucherRepoTx.Update(voucher); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *SolicitudService) Finalize(ctx context.Context, id string) error {
	return s.repo.WithContext(ctx).GetDB().Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)
		voucherRepoTx := s.voucherRepo.WithTx(tx)

		solicitud, err := repoTx.FindByID(id)
		if err != nil {
			return err
		}

		estadoFinalizado := "FINALIZADO"
		if err := repoTx.UpdateStatus(id, estadoFinalizado); err != nil {
			return err
		}

		if solicitud.VoucherID != nil {
			voucher, err := voucherRepoTx.FindByID(*solicitud.VoucherID)
			if err == nil && voucher != nil {
				estadoUsado := "USADO"
				voucher.EstadoVoucherCodigo = estadoUsado
				if err := voucherRepoTx.Update(voucher); err != nil {
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
