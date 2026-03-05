package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type PasajeRepository struct {
	db *gorm.DB
}

func NewPasajeRepository(db *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: db}
}

func (r *PasajeRepository) Create(ctx context.Context, pasaje *models.Pasaje) error {
	return r.db.WithContext(ctx).Create(pasaje).Error
}

func (r *PasajeRepository) FindBySolicitudID(ctx context.Context, solicitudID string) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	err := r.db.WithContext(ctx).Where("solicitud_id = ?", solicitudID).Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Pasaje{}, id).Error
}

func (r *PasajeRepository) FindByID(ctx context.Context, id string) (*models.Pasaje, error) {
	var pasaje models.Pasaje
	err := r.db.WithContext(ctx).Preload("EstadoPasaje").
		Preload("Agencia").
		Preload("Aerolinea").
		Preload("SolicitudItem").
		First(&pasaje, "id = ?", id).Error
	return &pasaje, err
}

func (r *PasajeRepository) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return r.db.WithContext(ctx).Save(pasaje).Error
}

func (r *PasajeRepository) WithTx(tx *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: tx}
}

func (r *PasajeRepository) WithContext(ctx context.Context) *PasajeRepository {
	return &PasajeRepository{db: r.db.WithContext(ctx)}
}

func (r *PasajeRepository) RunTransaction(fn func(repo *PasajeRepository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo, tx)
	})
}

func (r *PasajeRepository) FindByNumeroBoleto(ctx context.Context, numeroBoleto string) (*models.Pasaje, error) {
	var pasaje models.Pasaje
	err := r.db.WithContext(ctx).Where("numero_boleto = ?", numeroBoleto).First(&pasaje).Error
	return &pasaje, err
}
