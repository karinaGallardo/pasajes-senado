package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CategoriaCompensacionRepository struct {
	db *gorm.DB
}

func NewCategoriaCompensacionRepository(db *gorm.DB) *CategoriaCompensacionRepository {
	return &CategoriaCompensacionRepository{db: db}
}

func (r *CategoriaCompensacionRepository) FindAll() ([]models.CategoriaCompensacion, error) {
	var list []models.CategoriaCompensacion
	err := r.db.Order("departamento asc, tipo_senador asc").Find(&list).Error
	return list, err
}

func (r *CategoriaCompensacionRepository) Save(cat *models.CategoriaCompensacion) error {
	return r.db.Save(cat).Error
}

func (r *CategoriaCompensacionRepository) FindByDepartamentoAndTipo(dep, tipo string) (*models.CategoriaCompensacion, error) {
	var cat models.CategoriaCompensacion
	err := r.db.Where("departamento = ? AND tipo_senador = ?", dep, tipo).First(&cat).Error
	return &cat, err
}

func (r *CategoriaCompensacionRepository) Delete(id string) error {
	return r.db.Delete(&models.CategoriaCompensacion{}, "id = ?", id).Error
}
