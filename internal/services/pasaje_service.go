package services

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type PasajeService struct {
	repo       *repositories.PasajeRepository
	estadoRepo *repositories.EstadoPasajeRepository
	db         *gorm.DB
}

func NewPasajeService() *PasajeService {
	return &PasajeService{
		repo:       repositories.NewPasajeRepository(),
		estadoRepo: repositories.NewEstadoPasajeRepository(),
		db:         configs.DB,
	}
}

func (s *PasajeService) Create(ctx context.Context, pasaje *models.Pasaje) error {
	return s.repo.WithContext(ctx).Create(pasaje)
}

func (s *PasajeService) FindBySolicitudID(ctx context.Context, solicitudID string) ([]models.Pasaje, error) {
	return s.repo.WithContext(ctx).FindBySolicitudID(solicitudID)
}

func (s *PasajeService) Delete(ctx context.Context, id uint) error {
	return s.repo.WithContext(ctx).Delete(id)
}
func (s *PasajeService) FindByID(ctx context.Context, id string) (*models.Pasaje, error) {
	return s.repo.WithContext(ctx).FindByID(id)
}

func (s *PasajeService) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return s.repo.WithContext(ctx).Update(pasaje)
}

func (s *PasajeService) Reprogramar(ctx context.Context, pasajeAnteriorID string, nuevoPasaje *models.Pasaje) error {
	return s.repo.WithContext(ctx).GetDB().Transaction(func(tx *gorm.DB) error {
		repoTx := s.repo.WithTx(tx)
		estadoRepoTx := s.estadoRepo.WithTx(tx)

		pasajeAnterior, err := repoTx.FindByID(pasajeAnteriorID)
		if err != nil {
			return err
		}

		stateReprog := "REPROGRAMADO"
		if est, err := estadoRepoTx.FindByCodigo("REPROGRAMADO"); err == nil {
			stateReprog = est.Codigo
		}

		pasajeAnterior.EstadoPasajeCodigo = &stateReprog
		if err := repoTx.Update(pasajeAnterior); err != nil {
			return err
		}

		nuevoPasaje.SolicitudID = pasajeAnterior.SolicitudID
		nuevoPasaje.PasajeAnteriorID = &pasajeAnteriorID

		newState := "EMITIDO"
		if est, err := estadoRepoTx.FindByCodigo("EMITIDO"); err == nil {
			newState = est.Codigo
		}
		nuevoPasaje.EstadoPasajeCodigo = &newState

		return repoTx.Create(nuevoPasaje)
	})
}
