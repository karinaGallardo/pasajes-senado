package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type AgenciaRepository struct {
	db *gorm.DB
}

func NewAgenciaRepository(db *gorm.DB) *AgenciaRepository {
	return &AgenciaRepository{db: db}
}

func (r *AgenciaRepository) WithContext(ctx context.Context) *AgenciaRepository {
	return &AgenciaRepository{db: r.db.WithContext(ctx)}
}

func (r *AgenciaRepository) FindAllActive(ctx context.Context) ([]models.Agencia, error) {
	var agencias []models.Agencia
	err := r.db.WithContext(ctx).Where("estado = ?", true).Find(&agencias).Error
	return agencias, err
}

func (r *AgenciaRepository) FindAll(ctx context.Context) ([]models.Agencia, error) {
	var agencias []models.Agencia
	err := r.db.WithContext(ctx).Find(&agencias).Error
	return agencias, err
}

func (r *AgenciaRepository) Create(ctx context.Context, a *models.Agencia) error {
	return r.db.WithContext(ctx).Create(a).Error
}

func (r *AgenciaRepository) FindByID(ctx context.Context, id string) (*models.Agencia, error) {
	var a models.Agencia
	err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error
	return &a, err
}

func (r *AgenciaRepository) Update(ctx context.Context, a *models.Agencia) error {
	return r.db.WithContext(ctx).Save(a).Error
}

func (r *AgenciaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Agencia{}, "id = ?", id).Error
}
