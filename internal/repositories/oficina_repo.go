package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type OficinaRepository struct {
	db *gorm.DB
}

func NewOficinaRepository(db *gorm.DB) *OficinaRepository {
	return &OficinaRepository{db: db}
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
