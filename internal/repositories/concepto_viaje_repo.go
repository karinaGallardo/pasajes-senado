package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type ConceptoViajeRepository struct {
	db *gorm.DB
}

func NewConceptoViajeRepository() *ConceptoViajeRepository {
	return &ConceptoViajeRepository{db: configs.DB}
}

func (r *ConceptoViajeRepository) FindConceptos() ([]models.ConceptoViaje, error) {
	var conceptos []models.ConceptoViaje
	err := r.db.Preload("TiposSolicitud").Preload("TiposSolicitud.Ambitos").Find(&conceptos).Error
	return conceptos, err
}
