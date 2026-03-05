package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type RolRepository struct {
	db *gorm.DB
}

func NewRolRepository(db *gorm.DB) *RolRepository {
	return &RolRepository{db: db}
}

func (r *RolRepository) WithContext(ctx context.Context) *RolRepository {
	return &RolRepository{db: r.db.WithContext(ctx)}
}

func (r *RolRepository) FindAll(ctx context.Context) ([]models.Rol, error) {
	var roles []models.Rol
	err := r.db.WithContext(ctx).Order("nombre asc").Find(&roles).Error
	return roles, err
}

func (r *RolRepository) FindByCodigo(ctx context.Context, codigo string) (*models.Rol, error) {
	var rol models.Rol
	err := r.db.WithContext(ctx).Where("codigo = ?", codigo).First(&rol).Error
	return &rol, err
}
