package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type TipoItinerarioRepository struct {
	db *gorm.DB
}

func NewTipoItinerarioRepository() *TipoItinerarioRepository {
	return &TipoItinerarioRepository{db: configs.DB}
}

func (r *TipoItinerarioRepository) WithContext(ctx context.Context) *TipoItinerarioRepository {
	return &TipoItinerarioRepository{db: r.db.WithContext(ctx)}
}

func (r *TipoItinerarioRepository) FindByCodigo(codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := r.db.Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *TipoItinerarioRepository) FindByID(codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := r.db.First(&tipo, "codigo = ?", codigo).Error
	return &tipo, err
}

func (r *TipoItinerarioRepository) FindAll() ([]models.TipoItinerario, error) {
	var tipos []models.TipoItinerario
	err := r.db.Find(&tipos).Error
	return tipos, err
}

func (r *TipoItinerarioRepository) FirstOrCreate(tipo *models.TipoItinerario) error {
	return r.db.Where("codigo = ?", tipo.Codigo).FirstOrCreate(tipo).Error
}
