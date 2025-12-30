package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

)

type CatalogoService struct {
	conceptoRepo       *repositories.ConceptoViajeRepository
	tipoSolicitudRepo  *repositories.TipoSolicitudRepository
	ambitoRepo         *repositories.AmbitoViajeRepository
	tipoItinerarioRepo *repositories.TipoItinerarioRepository
}

func NewCatalogoService() *CatalogoService {
	return &CatalogoService{
		conceptoRepo:       repositories.NewConceptoViajeRepository(),
		tipoSolicitudRepo:  repositories.NewTipoSolicitudRepository(),
		ambitoRepo:         repositories.NewAmbitoViajeRepository(),
		tipoItinerarioRepo: repositories.NewTipoItinerarioRepository(),
	}
}

func (s *CatalogoService) GetAllConceptos() ([]models.ConceptoViaje, error) {
	return s.conceptoRepo.FindConceptos()
}

func (s *CatalogoService) GetTiposByConcepto(conceptoID string) ([]models.TipoSolicitud, error) {
	return s.tipoSolicitudRepo.FindByConceptoID(conceptoID)
}

func (s *CatalogoService) GetAmbitosByTipo(tipoID string) ([]models.AmbitoViaje, error) {
	return s.tipoSolicitudRepo.FindAmbitosByTipoID(tipoID)
}

func (s *CatalogoService) GetTipoByID(id string) (*models.TipoSolicitud, error) {
	return s.tipoSolicitudRepo.FindByID(id)
}

func (s *CatalogoService) GetTipoItinerarioByCodigo(codigo string) (*models.TipoItinerario, error) {
	return s.tipoItinerarioRepo.FindByCodigo(codigo)
}

func (s *CatalogoService) GetTiposItinerario() ([]models.TipoItinerario, error) {
	return s.tipoItinerarioRepo.FindAll()
}

func (s *CatalogoService) GetTiposSolicitud() ([]models.TipoSolicitud, error) {
	return s.tipoSolicitudRepo.FindAll()
}

func (s *CatalogoService) GetAmbitosViaje() ([]models.AmbitoViaje, error) {
	return s.ambitoRepo.FindAll()
}
