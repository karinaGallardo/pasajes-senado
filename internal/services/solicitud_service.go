package services

import (
	"errors"
	"log"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
	"strings"
)

type SolicitudService struct {
	repo              *repositories.SolicitudRepository
	tipoSolicitudRepo *repositories.TipoSolicitudRepository
	usuarioRepo       *repositories.UsuarioRepository
	cupoService       *CupoService
}

func NewSolicitudService() *SolicitudService {
	return &SolicitudService{
		repo:              repositories.NewSolicitudRepository(),
		tipoSolicitudRepo: repositories.NewTipoSolicitudRepository(),
		usuarioRepo:       repositories.NewUsuarioRepository(),
		cupoService:       NewCupoService(),
	}
}

func (s *SolicitudService) Create(solicitud *models.Solicitud, usuario *models.Usuario) error {
	tipoSolicitud, err := s.tipoSolicitudRepo.FindByID(solicitud.TipoSolicitudID)
	if err != nil {
		return errors.New("tipo de solicitud inválido o no encontrado")
	}
	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		beneficiary, err := s.usuarioRepo.FindByID(solicitud.UsuarioID)
		if err != nil {
			return errors.New("usuario beneficiario no encontrado")
		}

		esSenador := strings.HasPrefix(beneficiary.Tipo, "SENADOR")
		if !esSenador {
			return errors.New("solo los Senadores pueden recibir pasajes por derecho")
		}
	}

	solicitud.Estado = "SOLICITADO"

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

	return s.repo.Create(solicitud)
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

	if solicitud.TipoSolicitud != nil && solicitud.TipoSolicitud.ConceptoViaje != nil && solicitud.TipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		year := solicitud.FechaSalida.Year()
		month := int(solicitud.FechaSalida.Month())

		if err := s.cupoService.ProcesarConsumoPasaje(solicitud.UsuarioID, year, month); err != nil {
			log.Printf("Advertencia: No se pudo actualizar cupo para usuario %s: %v", solicitud.UsuarioID, err)
			return err
		}
	}

	solicitud.Estado = "APROBADO"
	return s.repo.Update(solicitud)
}

func (s *SolicitudService) Reject(id string) error {
	solicitud, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	solicitud.Estado = "RECHAZADO"
	return s.repo.Update(solicitud)
}

func (s *SolicitudService) Update(solicitud *models.Solicitud) error {
	return s.repo.Update(solicitud)
}
