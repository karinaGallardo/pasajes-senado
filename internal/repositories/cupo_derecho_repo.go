package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type CupoDerechoRepository struct {
	db *gorm.DB
}

func NewCupoDerechoRepository() *CupoDerechoRepository {
	return &CupoDerechoRepository{db: configs.DB}
}

func (r *CupoDerechoRepository) WithTx(tx *gorm.DB) *CupoDerechoRepository {
	return &CupoDerechoRepository{db: tx}
}

func (r *CupoDerechoRepository) WithContext(ctx context.Context) *CupoDerechoRepository {
	return &CupoDerechoRepository{db: r.db.WithContext(ctx)}
}

func (r *CupoDerechoRepository) RunTransaction(fn func(repo *CupoDerechoRepository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo, tx)
	})
}

func (r *CupoDerechoRepository) Create(cupo *models.CupoDerecho) error {
	return r.db.Create(cupo).Error
}

func (r *CupoDerechoRepository) Update(cupo *models.CupoDerecho) error {
	return r.db.Save(cupo).Error
}

func (r *CupoDerechoRepository) FindByTitularAndPeriodo(titularID string, gestion, mes int) (*models.CupoDerecho, error) {
	var cupo models.CupoDerecho
	err := r.db.Where("sen_titular_id = ? AND gestion = ? AND mes = ?", titularID, gestion, mes).First(&cupo).Error
	if err != nil {
		return nil, err
	}
	return &cupo, nil
}

func (r *CupoDerechoRepository) FindByPeriodo(gestion, mes int) ([]models.CupoDerecho, error) {
	var cupos []models.CupoDerecho
	err := r.db.Preload("SenTitular").Where("gestion = ? AND mes = ?", gestion, mes).Find(&cupos).Error
	return cupos, err
}

func (r *CupoDerechoRepository) FindByTitular(titularID string, gestion int) ([]models.CupoDerecho, error) {
	var cupos []models.CupoDerecho
	err := r.db.Where("sen_titular_id = ? AND gestion = ?", titularID, gestion).Order("mes asc").Find(&cupos).Error
	return cupos, err
}

func (r *CupoDerechoRepository) FindByID(id string) (*models.CupoDerecho, error) {
	var cupo models.CupoDerecho
	err := r.db.First(&cupo, "id = ?", id).Error
	return &cupo, err
}
