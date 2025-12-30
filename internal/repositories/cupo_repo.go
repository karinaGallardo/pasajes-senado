package repositories

import (
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

func (r *CupoRepository) Create(cupo *models.Cupo) error {
	return r.db.Create(cupo).Error
}

func (r *CupoRepository) Update(cupo *models.Cupo) error {
	return r.db.Save(cupo).Error
}

func (r *CupoRepository) FindByUsuarioAndPeriodo(usuarioID string, gestion, mes int) (*models.Cupo, error) {
	var cupo models.Cupo
	err := r.db.Where("usuario_id = ? AND gestion = ? AND mes = ?", usuarioID, gestion, mes).First(&cupo).Error
	if err != nil {
		return nil, err
	}
	return &cupo, nil
}

func (r *CupoRepository) FindByPeriodo(gestion, mes int) ([]models.Cupo, error) {
	var cupos []models.Cupo
	err := r.db.Preload("Usuario").Where("gestion = ? AND mes = ?", gestion, mes).Find(&cupos).Error
	return cupos, err
}

func (r *CupoRepository) FindByUsuario(usuarioID string, gestion int) ([]models.Cupo, error) {
	var cupos []models.Cupo
	err := r.db.Where("usuario_id = ? AND gestion = ?", usuarioID, gestion).Order("mes asc").Find(&cupos).Error
	return cupos, err
}
