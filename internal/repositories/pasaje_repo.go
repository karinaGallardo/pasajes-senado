package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type PasajeRepository struct {
	db *gorm.DB
}

func NewPasajeRepository() *PasajeRepository {
	return &PasajeRepository{db: configs.DB}
}

func (r *PasajeRepository) Create(pasaje *models.Pasaje) error {
	return r.db.Create(pasaje).Error
}

func (r *PasajeRepository) FindBySolicitudID(solicitudID string) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	err := r.db.Where("solicitud_id = ?", solicitudID).Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) Delete(id uint) error {
	return r.db.Delete(&models.Pasaje{}, id).Error
}

func (r *PasajeRepository) FindByID(id string) (*models.Pasaje, error) {
	var pasaje models.Pasaje
	err := r.db.Preload("EstadoPasaje").First(&pasaje, "id = ?", id).Error
	return &pasaje, err
}

func (r *PasajeRepository) Update(pasaje *models.Pasaje) error {
	return r.db.Save(pasaje).Error
}

func (r *PasajeRepository) WithTx(tx *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: tx}
}

func (r *PasajeRepository) WithContext(ctx context.Context) *PasajeRepository {
	return &PasajeRepository{db: r.db.WithContext(ctx)}
}

func (r *PasajeRepository) RunTransaction(fn func(repo *PasajeRepository) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo)
	})
}
