package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type EstadoPasajeRepository struct {
	db *gorm.DB
}

func NewEstadoPasajeRepository(db *gorm.DB) *EstadoPasajeRepository {
	return &EstadoPasajeRepository{db: db}
}

func (r *EstadoPasajeRepository) WithContext(ctx context.Context) *EstadoPasajeRepository {
	return &EstadoPasajeRepository{db: r.db.WithContext(ctx)}
}

func (r *EstadoPasajeRepository) FindByCodigo(ctx context.Context, codigo string) (*models.EstadoPasaje, error) {
	var estado models.EstadoPasaje
	err := r.db.WithContext(ctx).First(&estado, "codigo = ?", codigo).Error
	return &estado, err
}

func (r *EstadoPasajeRepository) FindAll(ctx context.Context) ([]models.EstadoPasaje, error) {
	var estados []models.EstadoPasaje
	err := r.db.WithContext(ctx).Find(&estados).Error
	return estados, err
}

func (r *EstadoPasajeRepository) WithTx(tx *gorm.DB) *EstadoPasajeRepository {
	return &EstadoPasajeRepository{db: tx}
}
