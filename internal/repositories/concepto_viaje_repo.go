package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type ConceptoViajeRepository struct {
	db *gorm.DB
}

func NewConceptoViajeRepository(db *gorm.DB) *ConceptoViajeRepository {
	return &ConceptoViajeRepository{db: db}
}

func (r *ConceptoViajeRepository) FindConceptos() ([]models.ConceptoViaje, error) {
	var conceptos []models.ConceptoViaje
	err := r.db.Preload("TiposSolicitud").Preload("TiposSolicitud.Ambitos").Find(&conceptos).Error
	return conceptos, err
}
