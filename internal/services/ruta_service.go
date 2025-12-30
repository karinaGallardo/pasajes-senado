package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type RutaService struct {
	rutaRepo *repositories.RutaRepository
}

func NewRutaService() *RutaService {
	return &RutaService{
		rutaRepo: repositories.NewRutaRepository(),
	}
}

func (s *RutaService) Create(tramo, sigla, ambito string) (*models.Ruta, error) {
	newRuta := &models.Ruta{
		Tramo:    tramo,
		Sigla:    sigla,
		NacInter: ambito,
	}
	err := s.rutaRepo.Create(newRuta)
	return newRuta, err
}

func (s *RutaService) GetAll() ([]models.Ruta, error) {
	return s.rutaRepo.FindAll()
}

func (s *RutaService) AssignContract(rutaID, aerolineaID string, monto float64) error {
	contrato := models.RutaContrato{
		RutaID:           rutaID,
		AerolineaID:      aerolineaID,
		MontoReferencial: monto,
	}
	return s.rutaRepo.AssignContract(&contrato)
}

func (s *RutaService) GetContractsByRuta(rutaID string) ([]models.RutaContrato, error) {
	return s.rutaRepo.GetContractsByRuta(rutaID)
}
