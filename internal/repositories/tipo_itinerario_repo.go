package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type TipoItinerarioRepository struct {
	db *gorm.DB
}

func NewTipoItinerarioRepository(db *gorm.DB) *TipoItinerarioRepository {
	return &TipoItinerarioRepository{db: db}
}

func (r *TipoItinerarioRepository) WithContext(ctx context.Context) *TipoItinerarioRepository {
	return &TipoItinerarioRepository{db: r.db.WithContext(ctx)}
}

func (r *TipoItinerarioRepository) FindByCodigo(ctx context.Context, codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := r.db.WithContext(ctx).Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *TipoItinerarioRepository) FindByID(ctx context.Context, codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := r.db.WithContext(ctx).First(&tipo, "codigo = ?", codigo).Error
	return &tipo, err
}

func (r *TipoItinerarioRepository) FindAll(ctx context.Context) ([]models.TipoItinerario, error) {
	var tipos []models.TipoItinerario
	err := r.db.WithContext(ctx).Find(&tipos).Error
	return tipos, err
}

func (r *TipoItinerarioRepository) FirstOrCreate(ctx context.Context, tipo *models.TipoItinerario) error {
	return r.db.WithContext(ctx).Where("codigo = ?", tipo.Codigo).FirstOrCreate(tipo).Error
}
