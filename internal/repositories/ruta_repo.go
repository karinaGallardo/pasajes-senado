package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type RutaRepository struct {
	db *gorm.DB
}

func NewRutaRepository() *RutaRepository {
	return &RutaRepository{db: configs.DB}
}

func (r *RutaRepository) FindAll() ([]models.Ruta, error) {
	var rutas []models.Ruta
	err := r.db.Preload("Origen").Preload("Destino").Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Create(ruta *models.Ruta) error {
	return r.db.Create(ruta).Error
}

func (r *RutaRepository) FindByID(id string) (*models.Ruta, error) {
	var ruta models.Ruta
	err := r.db.First(&ruta, "id = ?", id).Error
	return &ruta, err
}

func (r *RutaRepository) AssignContract(contrato *models.RutaContrato) error {
	return r.db.Create(contrato).Error
}

func (r *RutaRepository) GetContractsByRuta(rutaID string) ([]models.RutaContrato, error) {
	var contratos []models.RutaContrato
	err := r.db.Preload("Aerolinea").Where("ruta_id = ?", rutaID).Find(&contratos).Error
	return contratos, err
}
