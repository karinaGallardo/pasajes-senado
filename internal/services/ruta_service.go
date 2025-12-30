package services

import (
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"

	"gorm.io/gorm"
)

type RutaService struct {
	db       *gorm.DB
	rutaRepo *repositories.RutaRepository
}

func NewRutaService(db *gorm.DB) *RutaService {
	return &RutaService{
		db:       db,
		rutaRepo: repositories.NewRutaRepository(db),
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
	return s.db.Create(&contrato).Error
}

func (s *RutaService) GetContractsByRuta(rutaID string) ([]models.RutaContrato, error) {
	var contratos []models.RutaContrato
	err := s.db.Preload("Aerolinea").Where("ruta_id = ?", rutaID).Find(&contratos).Error
	return contratos, err
}
