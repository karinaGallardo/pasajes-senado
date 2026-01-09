package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type CupoRepository struct {
	db *gorm.DB
}

func NewCupoRepository() *CupoRepository {
	return &CupoRepository{db: configs.DB}
}

func (r *CupoRepository) WithTx(tx *gorm.DB) *CupoRepository {
	return &CupoRepository{db: tx}
}

func (r *CupoRepository) WithContext(ctx context.Context) *CupoRepository {
	return &CupoRepository{db: r.db.WithContext(ctx)}
}

func (r *CupoRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *CupoRepository) Create(cupo *models.Cupo) error {
	return r.db.Create(cupo).Error
}

func (r *CupoRepository) Update(cupo *models.Cupo) error {
	return r.db.Save(cupo).Error
}

func (r *CupoRepository) FindByTitularAndPeriodo(titularID string, gestion, mes int) (*models.Cupo, error) {
	var cupo models.Cupo
	err := r.db.Where("senador_id = ? AND gestion = ? AND mes = ?", titularID, gestion, mes).First(&cupo).Error
	if err != nil {
		return nil, err
	}
	return &cupo, nil
}

func (r *CupoRepository) FindByPeriodo(gestion, mes int) ([]models.Cupo, error) {
	var cupos []models.Cupo
	err := r.db.Preload("Senador").Where("gestion = ? AND mes = ?", gestion, mes).Find(&cupos).Error
	return cupos, err
}

func (r *CupoRepository) FindByTitular(titularID string, gestion int) ([]models.Cupo, error) {
	var cupos []models.Cupo
	err := r.db.Where("senador_id = ? AND gestion = ?", titularID, gestion).Order("mes asc").Find(&cupos).Error
	return cupos, err
}
