package services

import (
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
)

type SolicitudService struct {
	repo              *repositories.SolicitudRepository
	tipoSolicitudRepo *repositories.TipoSolicitudRepository
	usuarioRepo       *repositories.UsuarioRepository
}

func NewSolicitudService() *SolicitudService {
	return &SolicitudService{
		repo:              repositories.NewSolicitudRepository(),
		tipoSolicitudRepo: repositories.NewTipoSolicitudRepository(),
		usuarioRepo:       repositories.NewUsuarioRepository(),
	}
}

func (s *SolicitudService) Create(solicitud *models.Solicitud, usuario *models.Usuario, voucherID string) error {
	tipoSolicitud, err := s.tipoSolicitudRepo.FindByID(solicitud.TipoSolicitudID)
	if err != nil {
		return errors.New("tipo de solicitud inválido o no encontrado")
	}

	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		beneficiary, err := s.usuarioRepo.FindByID(solicitud.UsuarioID)
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
		exists, _ := s.repo.ExistsByCodigo(generated)
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

	if err := s.repo.Create(solicitud); err != nil {
		return err
	}

	return nil
}

func (s *SolicitudService) FindAll() ([]models.Solicitud, error) {
	return s.repo.FindAll()
}

func (s *SolicitudService) FindByID(id string) (*models.Solicitud, error) {
	return s.repo.FindByID(id)
}

func (s *SolicitudService) Approve(id string) error {
	solicitud, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	estadoAprobado := "APROBADO"
	if err := s.repo.UpdateStatus(id, estadoAprobado); err != nil {
		return err
	}

	if solicitud.VoucherID != nil {
		voucherRepo := repositories.NewAsignacionVoucherRepository()
		voucher, err := voucherRepo.FindByID(*solicitud.VoucherID)
		if err == nil && voucher != nil {
			estadoReservado := "RESERVADO"
			voucher.EstadoVoucherCodigo = estadoReservado
			voucherRepo.Update(voucher)
		}
	}

	return nil
}

func (s *SolicitudService) Finalize(id string) error {
	solicitud, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}

	estadoFinalizado := "FINALIZADO"
	if err := s.repo.UpdateStatus(id, estadoFinalizado); err != nil {
		return err
	}

	if solicitud.VoucherID != nil {
		voucherRepo := repositories.NewAsignacionVoucherRepository()
		voucher, err := voucherRepo.FindByID(*solicitud.VoucherID)
		if err == nil && voucher != nil {
			estadoUsado := "USADO"
			voucher.EstadoVoucherCodigo = estadoUsado
			voucherRepo.Update(voucher)
		}
	}

	return nil
}

func (s *SolicitudService) Reject(id string) error {
	return s.repo.UpdateStatus(id, "RECHAZADO")
}

func (s *SolicitudService) Update(solicitud *models.Solicitud) error {
	return s.repo.Update(solicitud)
}

func (s *SolicitudService) Delete(id string) error {
	return s.repo.Delete(id)
}
