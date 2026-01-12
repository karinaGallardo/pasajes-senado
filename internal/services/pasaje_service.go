package services

import (
	"context"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/utils"
)

type PasajeService struct {
	repo       *repositories.PasajeRepository
	estadoRepo *repositories.EstadoPasajeRepository
}

func NewPasajeService() *PasajeService {
	return &PasajeService{
		repo:       repositories.NewPasajeRepository(),
		estadoRepo: repositories.NewEstadoPasajeRepository(),
	}
}

func (s *PasajeService) Create(ctx context.Context, solicitudID string, req dtos.CreatePasajeRequest, filePath string) (*models.Pasaje, error) {
	costo := utils.ParseFloat(req.Costo)
	fechaVuelo := utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	pasaje := &models.Pasaje{
		SolicitudID:   solicitudID,
		AerolineaID:   aerolineaID,
		AgenciaID:     &req.AgenciaID,
		NumeroVuelo:   req.NumeroVuelo,
		Ruta:          req.Ruta,
		FechaVuelo:    fechaVuelo,
		CodigoReserva: req.CodigoReserva,
		NumeroBoleto:  req.NumeroBoleto,
		NumeroFactura: req.NumeroFactura,
		Glosa:         req.Glosa,
		Costo:         costo,
		Archivo:       filePath,
	}

	if err := s.repo.WithContext(ctx).Create(pasaje); err != nil {
		return nil, err
	}
	return pasaje, nil
}

func (s *PasajeService) GetBySolicitudID(ctx context.Context, solicitudID string) ([]models.Pasaje, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *PasajeService) Delete(ctx context.Context, id uint) error {
	return s.repo.WithContext(ctx).Delete(id)
}
func (s *PasajeService) GetByID(ctx context.Context, id string) (*models.Pasaje, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *PasajeService) UpdateFromRequest(ctx context.Context, req dtos.UpdatePasajeRequest, archivo string, paseAbordo string) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(req.ID)
	if err != nil {
		return err
	}

	pasaje.NumeroVuelo = req.NumeroVuelo
	pasaje.Ruta = req.Ruta
	pasaje.NumeroBoleto = req.NumeroBoleto
	pasaje.NumeroFactura = req.NumeroFactura
	pasaje.CodigoReserva = req.CodigoReserva
	pasaje.Glosa = req.Glosa

	if req.AerolineaID != "" {
		pasaje.AerolineaID = &req.AerolineaID
	}

	pasaje.Costo = utils.ParseFloat(req.Costo)

	pasaje.FechaVuelo = utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	if archivo != "" {
		pasaje.Archivo = archivo
	}
	if paseAbordo != "" {
		pasaje.ArchivoPaseAbordo = paseAbordo
	}

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) Reprogramar(ctx context.Context, req dtos.ReprogramarPasajeRequest, filePath string) error {
	stateReprog := "REPROGRAMADO"
	newState := "EMITIDO"

	costo := utils.ParseFloat(req.Costo)
	penalidad := utils.ParseFloat(req.CostoPenalidad)
	fecha := utils.ParseDate("2006-01-02T15:04", req.FechaVuelo)

	var aerolineaID *string
	if req.AerolineaID != "" {
		aerolineaID = &req.AerolineaID
	}

	return s.repo.WithContext(ctx).RunTransaction(func(repoTx *repositories.PasajeRepository) error {
		pasajeAnterior, err := repoTx.FindByID(req.PasajeAnteriorID)
		if err != nil {
			return err
		}

		pasajeAnterior.EstadoPasajeCodigo = &stateReprog
		if err := repoTx.Update(pasajeAnterior); err != nil {
			return err
		}

		nuevoPasaje := &models.Pasaje{
			SolicitudID:        pasajeAnterior.SolicitudID,
			PasajeAnteriorID:   &req.PasajeAnteriorID,
			EstadoPasajeCodigo: &newState,
			AerolineaID:        aerolineaID,
			AgenciaID:          &req.AgenciaID,
			NumeroVuelo:        req.NumeroVuelo,
			Ruta:               req.Ruta,
			FechaVuelo:         fecha,
			NumeroBoleto:       req.NumeroBoleto,
			Costo:              costo,
			CostoPenalidad:     penalidad,
			Archivo:            filePath,
			Glosa:              req.Glosa,
			NumeroFactura:      req.NumeroFactura,
			CodigoReserva:      req.CodigoReserva,
		}

		return repoTx.Create(nuevoPasaje)
	})
}
func (s *PasajeService) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) DevolverPasaje(ctx context.Context, req dtos.DevolverPasajeRequest) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(req.PasajeID)
	if err != nil {
		return err
	}

	costoPenalidad := utils.ParseFloat(req.CostoPenalidad)
	pasaje.EstadoPasajeCodigo = utils.Ptr("DEVUELTO")

	if pasaje.Glosa != "" {
		pasaje.Glosa += " | Devolución: " + req.Glosa
	} else {
		pasaje.Glosa = "Devolución: " + req.Glosa
	}
	pasaje.CostoPenalidad = costoPenalidad

	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) UpdateStatus(ctx context.Context, pasajeID string, status string, archivoPase string) error {
	pasaje, err := s.repo.WithContext(ctx).FindByID(pasajeID)
	if err != nil {
		return err
	}

	pasaje.EstadoPasajeCodigo = &status
	if status == "VALIDANDO_USO" && archivoPase != "" {
		pasaje.ArchivoPaseAbordo = archivoPase
	}

	return s.repo.WithContext(ctx).Update(pasaje)
}
