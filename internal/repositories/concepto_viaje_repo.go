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
func (r *ConceptoViajeRepository) FindByCodigo(codigo string) (*models.ConceptoViaje, error) {
	var concepto models.ConceptoViaje
	err := r.db.Where("codigo = ?", codigo).First(&concepto).Error
	return &concepto, err
}
