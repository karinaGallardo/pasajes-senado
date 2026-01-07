package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type RutaService struct {
	rutaRepo    *repositories.RutaRepository
	destinoRepo *repositories.DestinoRepository
}

func NewRutaService() *RutaService {
	return &RutaService{
		rutaRepo:    repositories.NewRutaRepository(),
		destinoRepo: repositories.NewDestinoRepository(),
	}
}

func (s *RutaService) Create(origenIATA string, escalasIATA []string, destinoIATA string) (*models.Ruta, error) {
	origen, err := s.destinoRepo.FindByIATA(origenIATA)
	if err != nil {
		return nil, err
	}
	destino, err := s.destinoRepo.FindByIATA(destinoIATA)
	if err != nil {
		return nil, err
	}

	var escalas []models.Destino
	for _, code := range escalasIATA {
		if code == "" {
			continue
		}
		e, err := s.destinoRepo.FindByIATA(code)
		if err != nil {
			return nil, err
		}
		escalas = append(escalas, *e)
	}

	tramo := origen.Ciudad
	sigla := origen.IATA

	for _, e := range escalas {
		tramo += " - " + e.Ciudad
		sigla += "-" + e.IATA
	}

	tramo += " - " + destino.Ciudad
	sigla += "-" + destino.IATA

	ambito := "INTERNACIONAL"
	ubicacion := "EXTERIOR"

	isNacional := func(d *models.Destino) bool {
		return d.AmbitoCodigo == "NACIONAL"
	}

	allNacional := isNacional(origen) && isNacional(destino)
	for _, e := range escalas {
		if !isNacional(&e) {
			allNacional = false
			break
		}
	}

	if allNacional {
		ambito = "NACIONAL"
		ubicacion = "INTERIOR"
	}

	newRuta := &models.Ruta{
		Tramo:       tramo,
		Sigla:       sigla,
		NacInter:    ambito,
		Ubicacion:   ubicacion,
		OrigenIATA:  origenIATA,
		DestinoIATA: destinoIATA,
	}

	for i, e := range escalas {
		newRuta.Escalas = append(newRuta.Escalas, models.RutaEscala{
			DestinoIATA: e.IATA,
			Orden:       i + 1,
		})
	}

	err = s.rutaRepo.Create(newRuta)
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
