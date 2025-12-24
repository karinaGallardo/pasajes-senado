package services

import (
	"errors"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"strings"
)

type SolicitudService struct {
	repo         *repositories.SolicitudRepository
	catalogoRepo *repositories.CatalogoRepository
}

func NewSolicitudService() *SolicitudService {
	return &SolicitudService{
		repo:         repositories.NewSolicitudRepository(),
		catalogoRepo: repositories.NewCatalogoRepository(),
	}
}

func (s *SolicitudService) Create(solicitud *models.Solicitud, usuario *models.Usuario) error {
	tipoSolicitud, err := s.catalogoRepo.FindTipoSolicitudByID(solicitud.TipoSolicitudID)
	if err != nil {
		return errors.New("tipo de solicitud inválido o no encontrado")
	}
	if tipoSolicitud.ConceptoViaje != nil && tipoSolicitud.ConceptoViaje.Codigo == "DERECHO" {
		esSenador := strings.HasPrefix(usuario.Tipo, "SENADOR")
		if !esSenador {
			return errors.New("solo los Senadores pueden solicitar pasajes por derecho")
		}
	}

	solicitud.Estado = "SOLICITADO"

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
