package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type TipoSolicitudService struct {
	repo *repositories.TipoSolicitudRepository
}

func NewTipoSolicitudService() *TipoSolicitudService {
	return &TipoSolicitudService{
		repo: repositories.NewTipoSolicitudRepository(),
	}
}

func (s *TipoSolicitudService) GetByConcepto(conceptoID string) ([]models.TipoSolicitud, error) {
	return s.repo.FindByConceptoID(conceptoID)
}

func (s *TipoSolicitudService) GetAmbitosByTipo(tipoID string) ([]models.AmbitoViaje, error) {
	return s.repo.FindAmbitosByTipoID(tipoID)
}

func (s *TipoSolicitudService) GetByID(id string) (*models.TipoSolicitud, error) {
	return s.repo.FindByID(id)
}

func (s *TipoSolicitudService) GetAll() ([]models.TipoSolicitud, error) {
	return s.repo.FindAll()
}

func (s *TipoSolicitudService) GetByCodigo(codigo string) (*models.TipoSolicitud, error) {
	return s.repo.FindByCodigo(codigo)
}

func (s *TipoSolicitudService) GetByCodigoAndAmbito(tipoCodigo, ambitoCodigo string) (*models.TipoSolicitud, *models.AmbitoViaje, error) {
	return s.repo.FindByCodigoAndAmbito(tipoCodigo, ambitoCodigo)
}
