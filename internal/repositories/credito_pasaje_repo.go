package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CreditoPasajeRepository struct {
	db *gorm.DB
}

func NewCreditoPasajeRepository(db *gorm.DB) *CreditoPasajeRepository {
	return &CreditoPasajeRepository{db: db}
}

func (r *CreditoPasajeRepository) WithTx(tx *gorm.DB) *CreditoPasajeRepository {
	return &CreditoPasajeRepository{db: tx}
}

func (r *CreditoPasajeRepository) FindByDescargoID(ctx context.Context, descargoID string) ([]models.CreditoPasaje, error) {
	var creditos []models.CreditoPasaje
	err := r.db.WithContext(ctx).
		Preload("Descargo").
		Where("descargo_id = ?", descargoID).
		Find(&creditos).Error
	return creditos, err
}

func (r *CreditoPasajeRepository) Create(ctx context.Context, credito *models.CreditoPasaje) error {
	return r.db.WithContext(ctx).Create(credito).Error
}

func (r *CreditoPasajeRepository) FindByUsuarioID(ctx context.Context, usuarioID string) ([]models.CreditoPasaje, error) {
	var creditos []models.CreditoPasaje
	err := r.db.WithContext(ctx).
		Preload("Descargo").
		Preload("Tramo").
		Where("usuario_id = ? AND estado = ?", usuarioID, models.EstadoCreditoDisponible).
		Order("created_at DESC").
		Find(&creditos).Error
	return creditos, err
}

func (r *CreditoPasajeRepository) FindByID(ctx context.Context, id string) (*models.CreditoPasaje, error) {
	var credito models.CreditoPasaje
	err := r.db.WithContext(ctx).
		Preload("Usuario").
		Preload("Descargo").
		Preload("Tramo").
		First(&credito, "id = ?", id).Error
	return &credito, err
}

func (r *CreditoPasajeRepository) Update(ctx context.Context, credito *models.CreditoPasaje) error {
	return r.db.WithContext(ctx).Save(credito).Error
}

func (r *CreditoPasajeRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.CreditoPasaje{}, "id = ?", id).Error
}
