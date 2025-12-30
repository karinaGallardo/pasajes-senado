package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type RolRepository struct {
	db *gorm.DB
}

func NewRolRepository() *RolRepository {
	return &RolRepository{db: configs.DB}
}

func (r *RolRepository) FindAll() ([]models.Rol, error) {
	var roles []models.Rol
	err := r.db.Order("nombre asc").Find(&roles).Error
	return roles, err
}
