package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type TipoItinerarioRepository struct {
	db *gorm.DB
}

func NewTipoItinerarioRepository(db *gorm.DB) *TipoItinerarioRepository {
	return &TipoItinerarioRepository{db: db}
}

func (r *TipoItinerarioRepository) FindByCodigo(codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := r.db.Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *TipoItinerarioRepository) FindAll() ([]models.TipoItinerario, error) {
	var tipos []models.TipoItinerario
	err := r.db.Find(&tipos).Error
	return tipos, err
}
