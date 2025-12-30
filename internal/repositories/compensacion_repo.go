package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CompensacionRepository struct {
	db *gorm.DB
}

func NewCompensacionRepository(db *gorm.DB) *CompensacionRepository {
	return &CompensacionRepository{db: db}
}

func (r *CompensacionRepository) Create(comp *models.Compensacion) error {
	return r.db.Create(comp).Error
}

func (r *CompensacionRepository) FindAll() ([]models.Compensacion, error) {
	var list []models.Compensacion
	err := r.db.Preload("Funcionario").Order("created_at desc").Find(&list).Error
	return list, err
}

func (r *CompensacionRepository) FindByID(id string) (*models.Compensacion, error) {
	var comp models.Compensacion
	err := r.db.Preload("Funcionario").First(&comp, "id = ?", id).Error
	return &comp, err
}

func (r *CompensacionRepository) Update(comp *models.Compensacion) error {
	return r.db.Save(comp).Error
}
