package repositories

import (
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type GeneroRepository struct {
	db *gorm.DB
}

func NewGeneroRepository() *GeneroRepository {
	return &GeneroRepository{db: configs.DB}
}

func (r *GeneroRepository) FirstOrCreate(codigo, nombre string) (*models.Genero, error) {
	var genero models.Genero
	err := r.db.Where(models.Genero{Codigo: codigo}).
		Attrs(models.Genero{Nombre: nombre}).
		FirstOrCreate(&genero).Error
	return &genero, err
}
