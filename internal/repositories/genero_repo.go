package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type GeneroRepository struct {
	db *gorm.DB
}

func NewGeneroRepository(db *gorm.DB) *GeneroRepository {
	return &GeneroRepository{db: db}
}

func (r *GeneroRepository) WithContext(ctx context.Context) *GeneroRepository {
	return &GeneroRepository{db: r.db.WithContext(ctx)}
}

func (r *GeneroRepository) FirstOrCreate(ctx context.Context, codigo, nombre string) (*models.Genero, error) {
	var genero models.Genero
	err := r.db.WithContext(ctx).Where(models.Genero{Codigo: codigo}).
		Attrs(models.Genero{Nombre: nombre}).
		FirstOrCreate(&genero).Error
	return &genero, err
}
