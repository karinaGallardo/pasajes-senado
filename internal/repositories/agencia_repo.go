package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type AgenciaRepository struct {
	db *gorm.DB
}

func NewAgenciaRepository() *AgenciaRepository {
	return &AgenciaRepository{db: configs.DB}
}

func (r *AgenciaRepository) WithContext(ctx context.Context) *AgenciaRepository {
	return &AgenciaRepository{db: r.db.WithContext(ctx)}
}

func (r *AgenciaRepository) FindAllActive() ([]models.Agencia, error) {
	var agencias []models.Agencia
	err := r.db.Where("estado = ?", true).Find(&agencias).Error
	return agencias, err
}

func (r *AgenciaRepository) FindAll() ([]models.Agencia, error) {
	var agencias []models.Agencia
	err := r.db.Find(&agencias).Error
	return agencias, err
}

func (r *AgenciaRepository) Create(a *models.Agencia) error {
	return r.db.Create(a).Error
}

func (r *AgenciaRepository) FindByID(id string) (*models.Agencia, error) {
	var a models.Agencia
	err := r.db.First(&a, "id = ?", id).Error
	return &a, err
}

func (r *AgenciaRepository) Save(a *models.Agencia) error {
	return r.db.Save(a).Error
}

func (r *AgenciaRepository) Delete(id string) error {
	return r.db.Delete(&models.Agencia{}, "id = ?", id).Error
}
