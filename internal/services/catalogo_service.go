package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type CatalogoService struct {
	repo *repositories.CatalogoRepository
}

func NewCatalogoService() *CatalogoService {
	return &CatalogoService{
		repo: repositories.NewCatalogoRepository(),
	}
}

func (s *CatalogoService) GetAllConceptos() ([]models.ConceptoViaje, error) {
	return s.repo.FindConceptos()
}

func (s *CatalogoService) GetTiposByConcepto(conceptoID string) ([]models.TipoSolicitud, error) {
	return s.repo.FindTiposByConceptoID(conceptoID)
}

func (s *CatalogoService) GetAmbitosByTipo(tipoID string) ([]models.AmbitoViaje, error) {
	return s.repo.FindAmbitosByTipoID(tipoID)
}

func (s *CatalogoService) GetTipoByID(id string) (*models.TipoSolicitud, error) {
	return s.repo.FindTipoSolicitudByID(id)
}

func (s *CatalogoService) GetTipoItinerarioByCodigo(codigo string) (*models.TipoItinerario, error) {
	return s.repo.FindTipoItinerarioByCodigo(codigo)
}

func (s *CatalogoService) GetTiposItinerario() ([]models.TipoItinerario, error) {
	return s.repo.FindAllTiposItinerario()
}

func (s *CatalogoService) GetTiposSolicitud() ([]models.TipoSolicitud, error) {
	return s.repo.FindAllTiposSolicitud()
}

func (s *CatalogoService) GetAmbitosViaje() ([]models.AmbitoViaje, error) {
	return s.repo.FindAllAmbitosViaje()
}
