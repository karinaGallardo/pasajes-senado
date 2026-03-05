package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type CompensacionRepository struct {
	db *gorm.DB
}

func NewCompensacionRepository(db *gorm.DB) *CompensacionRepository {
	return &CompensacionRepository{db: db}
}

func (r *CompensacionRepository) WithContext(ctx context.Context) *CompensacionRepository {
	return &CompensacionRepository{db: r.db.WithContext(ctx)}
}

func (r *CompensacionRepository) Create(ctx context.Context, comp *models.Compensacion) error {
	return r.db.WithContext(ctx).Create(comp).Error
}

func (r *CompensacionRepository) FindAll(ctx context.Context) ([]models.Compensacion, error) {
	var list []models.Compensacion
	err := r.db.WithContext(ctx).Preload("Funcionario").Order("created_at desc").Find(&list).Error
	return list, err
}

func (r *CompensacionRepository) FindByID(ctx context.Context, id string) (*models.Compensacion, error) {
	var comp models.Compensacion
	err := r.db.WithContext(ctx).Preload("Funcionario").First(&comp, "id = ?", id).Error
	return &comp, err
}

func (r *CompensacionRepository) Update(ctx context.Context, comp *models.Compensacion) error {
	return r.db.WithContext(ctx).Save(comp).Error
}
