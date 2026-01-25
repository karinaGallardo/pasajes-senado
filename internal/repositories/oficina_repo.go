package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type OficinaRepository struct {
	db *gorm.DB
}

func NewOficinaRepository() *OficinaRepository {
	return &OficinaRepository{db: configs.DB}
}

func (r *OficinaRepository) WithContext(ctx context.Context) *OficinaRepository {
	return &OficinaRepository{db: r.db.WithContext(ctx)}
}

func (r *OficinaRepository) FindAll() ([]models.Oficina, error) {
	var oficinas []models.Oficina
	err := r.db.Order("codigo asc").Find(&oficinas).Error
	return oficinas, err
}

func (r *OficinaRepository) Create(o *models.Oficina) error {
	return r.db.Create(o).Error
}

func (r *OficinaRepository) FindByID(id string) (*models.Oficina, error) {
	var oficina models.Oficina
	err := r.db.First(&oficina, "id = ?", id).Error
	return &oficina, err
}

func (r *OficinaRepository) Delete(id string) error {
	return r.db.Delete(&models.Oficina{}, "id = ?", id).Error
}

func (r *OficinaRepository) FindByDetalle(detalle string) (*models.Oficina, error) {
	var oficina models.Oficina
	err := r.db.Where("detalle ILIKE ?", detalle).First(&oficina).Error
	return &oficina, err
}
func (r *OficinaRepository) GetNextCodigo() (int, error) {
	var maxCodigo int
	err := r.db.Model(&models.Oficina{}).Select("COALESCE(MAX(codigo), 0)").Scan(&maxCodigo).Error
	return maxCodigo + 1, err
}
