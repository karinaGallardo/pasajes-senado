package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type RolRepository struct {
	db *gorm.DB
}

func NewRolRepository(db *gorm.DB) *RolRepository {
	return &RolRepository{db: db}
}

func (r *RolRepository) FindAll() ([]models.Rol, error) {
	var roles []models.Rol
	err := r.db.Order("nombre asc").Find(&roles).Error
	return roles, err
}
