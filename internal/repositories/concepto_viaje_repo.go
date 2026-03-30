package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type ConceptoViajeRepository struct {
	db *gorm.DB
}

func NewConceptoViajeRepository(db *gorm.DB) *ConceptoViajeRepository {
	return &ConceptoViajeRepository{db: db}
}

func (r *ConceptoViajeRepository) WithContext(ctx context.Context) *ConceptoViajeRepository {
	return &ConceptoViajeRepository{db: r.db.WithContext(ctx)}
}

func (r *ConceptoViajeRepository) FindConceptos(ctx context.Context) ([]models.ConceptoViaje, error) {
	var conceptos []models.ConceptoViaje
	err := r.db.WithContext(ctx).Preload("TiposSolicitud").Preload("TiposSolicitud.Ambitos").Find(&conceptos).Error
	return conceptos, err
}
func (r *ConceptoViajeRepository) FindByCodigo(ctx context.Context, codigo string) (*models.ConceptoViaje, error) {
	var concepto models.ConceptoViaje
	err := r.db.WithContext(ctx).Where("codigo = ?", codigo).First(&concepto).Error
	return &concepto, err
}
