package services

import (
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

func (s *PasajeService) Create(pasaje *models.Pasaje) error {
	return s.repo.Create(pasaje)
}

func (s *PasajeService) FindBySolicitudID(solicitudID string) ([]models.Pasaje, error) {
	return s.repo.FindBySolicitudID(solicitudID)
}

func (s *PasajeService) Delete(id uint) error {
	return s.repo.Delete(id)
}
func (s *PasajeService) FindByID(id string) (*models.Pasaje, error) {
	return s.repo.FindByID(id)
}

func (s *PasajeService) Update(pasaje *models.Pasaje) error {
	return s.repo.Update(pasaje)
}

func (s *PasajeService) Reprogramar(pasajeAnteriorID string, nuevoPasaje *models.Pasaje) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var pasajeAnterior models.Pasaje
		if err := tx.Preload("EstadoPasaje").First(&pasajeAnterior, "id = ?", pasajeAnteriorID).Error; err != nil {
			return err
		}

		stateReprog := "REPROGRAMADO"
		var estadoReprog models.EstadoPasaje
		if err := tx.Where("codigo = ?", "REPROGRAMADO").First(&estadoReprog).Error; err == nil {
			stateReprog = estadoReprog.Codigo
		}

		if err := tx.Model(&models.Pasaje{}).Where("id = ?", pasajeAnteriorID).Update("estado_pasaje_codigo", stateReprog).Error; err != nil {
			return err
		}

		nuevoPasaje.SolicitudID = pasajeAnterior.SolicitudID

		nuevoPasaje.PasajeAnteriorID = &pasajeAnteriorID

		newState := "EMITIDO"
		var estadoEmitido models.EstadoPasaje
		if err := tx.Where("codigo = ?", "EMITIDO").First(&estadoEmitido).Error; err == nil {
			newState = estadoEmitido.Codigo
		}
		nuevoPasaje.EstadoPasajeCodigo = &newState

		return tx.Create(nuevoPasaje).Error
	})
}
