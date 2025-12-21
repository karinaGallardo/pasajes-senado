package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type RolRepository struct{}

func NewRolRepository() *RolRepository {
	return &RolRepository{}
}

func (r *RolRepository) FindAll() ([]models.Rol, error) {
	var roles []models.Rol
	err := configs.DB.Order("nombre asc").Find(&roles).Error
	return roles, err
}
