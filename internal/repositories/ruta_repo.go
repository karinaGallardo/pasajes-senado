package repositories

import (
	"context"
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

func (r *RutaRepository) WithContext(ctx context.Context) *RutaRepository {
	return &RutaRepository{db: r.db.WithContext(ctx)}
}

func (r *RutaRepository) FindAll() ([]models.Ruta, error) {
	var rutas []models.Ruta
	err := r.db.Preload("Origen").Preload("Destino").Preload("Contratos.Aerolinea").Find(&rutas).Error
	return rutas, err
}

func (r *RutaRepository) Create(ruta *models.Ruta) error {
	return r.db.Create(ruta).Error
}

func (r *RutaRepository) FindByID(id string) (*models.Ruta, error) {
	var ruta models.Ruta
	err := r.db.Preload("Contratos.Aerolinea").First(&ruta, "id = ?", id).Error
	return &ruta, err
}

func (r *RutaRepository) AssignContract(contrato *models.RutaContrato) error {
	var existing models.RutaContrato
	err := r.db.Where("ruta_id = ? AND aerolinea_id = ?", contrato.RutaID, contrato.AerolineaID).First(&existing).Error
	if err == nil {
		// Update existing
		return r.db.Model(&existing).Update("monto_referencial", contrato.MontoReferencial).Error
	}
	return r.db.Create(contrato).Error
}

func (r *RutaRepository) GetContractsByRuta(rutaID string) ([]models.RutaContrato, error) {
	var contratos []models.RutaContrato
	err := r.db.Preload("Aerolinea").Where("ruta_id = ?", rutaID).Find(&contratos).Error
	return contratos, err
}

func (r *RutaRepository) DeleteContract(id string) error {
	return r.db.Delete(&models.RutaContrato{}, "id = ?", id).Error
}
