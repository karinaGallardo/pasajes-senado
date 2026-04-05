package services

import (
	"context"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
)

type RutaService struct {
	rutaRepo    *repositories.RutaRepository
	destinoRepo *repositories.DestinoRepository
}

func NewRutaService(rutaRepo *repositories.RutaRepository, destinoRepo *repositories.DestinoRepository) *RutaService {
	return &RutaService{
		rutaRepo:    rutaRepo,
		destinoRepo: destinoRepo,
	}
}

func (s *RutaService) Create(ctx context.Context, origenIATA string, escalasIATA []string, destinoIATA string) (*models.Ruta, error) {
	origen, err := s.destinoRepo.FindByIATA(ctx, origenIATA)
	if err != nil {
		return nil, err
	}
	destino, err := s.destinoRepo.FindByIATA(ctx, destinoIATA)
	if err != nil {
		return nil, err
	}

	var escalas []models.Destino
	for _, code := range escalasIATA {
		if code == "" {
			continue
		}
		e, err := s.destinoRepo.FindByIATA(ctx, code)
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

	for _, e := range escalas {
		newRuta.Escalas = append(newRuta.Escalas, models.RutaEscala{
			DestinoIATA: e.IATA,
		})
	}

	err = s.rutaRepo.Create(ctx, newRuta)
	return newRuta, err
}

func (s *RutaService) GetAll(ctx context.Context) ([]models.Ruta, error) {
	return s.rutaRepo.FindAll(ctx)
}

func (s *RutaService) Search(ctx context.Context, query string, onlyAtomic bool) ([]models.Ruta, error) {
	return s.rutaRepo.Search(ctx, query, onlyAtomic)
}

func (s *RutaService) GetFaresMap(rutas []models.Ruta) map[string]map[string]float64 {
	fares := make(map[string]map[string]float64)
	for _, r := range rutas {
		m := make(map[string]float64)
		for _, c := range r.Contratos {
			m[c.AerolineaID] = c.MontoReferencial
		}
		fares[r.ID] = m
	}
	return fares
}

func (s *RutaService) GetByID(ctx context.Context, id string) (*models.Ruta, error) {
	return s.rutaRepo.FindByID(ctx, id)
}

func (s *RutaService) AssignContract(ctx context.Context, rutaID, aerolineaID string, monto float64) error {
	contrato := models.RutaContrato{
		RutaID:           rutaID,
		AerolineaID:      aerolineaID,
		MontoReferencial: monto,
	}
	return s.rutaRepo.AssignContract(ctx, &contrato)
}

func (s *RutaService) GetContractsByRuta(ctx context.Context, rutaID string) ([]models.RutaContrato, error) {
	return s.rutaRepo.GetContractsByRuta(ctx, rutaID)
}

func (s *RutaService) RemoveContract(ctx context.Context, id string) error {
	return s.rutaRepo.DeleteContract(ctx, id)
}
