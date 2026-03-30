package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type OficinaRepository struct {
	db *gorm.DB
}

func NewOficinaRepository(db *gorm.DB) *OficinaRepository {
	return &OficinaRepository{db: db}
}

func (r *OficinaRepository) WithContext(ctx context.Context) *OficinaRepository {
	return &OficinaRepository{db: r.db.WithContext(ctx)}
}

func (r *OficinaRepository) FindAll(ctx context.Context) ([]models.Oficina, error) {
	var oficinas []models.Oficina
	err := r.db.WithContext(ctx).Order("codigo asc").Find(&oficinas).Error
	return oficinas, err
}

func (r *OficinaRepository) Create(ctx context.Context, o *models.Oficina) error {
	return r.db.WithContext(ctx).Create(o).Error
}

func (r *OficinaRepository) FindByID(ctx context.Context, id string) (*models.Oficina, error) {
	var oficina models.Oficina
	err := r.db.WithContext(ctx).First(&oficina, "id = ?", id).Error
	return &oficina, err
}

func (r *OficinaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Oficina{}, "id = ?", id).Error
}

func (r *OficinaRepository) FindByDetalle(ctx context.Context, detalle string) (*models.Oficina, error) {
	var oficina models.Oficina
	err := r.db.WithContext(ctx).Where("detalle ILIKE ?", detalle).First(&oficina).Error
	return &oficina, err
}
func (r *OficinaRepository) GetNextCodigo(ctx context.Context) (int, error) {
	var maxCodigo int
	err := r.db.WithContext(ctx).Model(&models.Oficina{}).Select("COALESCE(MAX(codigo), 0)").Scan(&maxCodigo).Error
	return maxCodigo + 1, err
}
