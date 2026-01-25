package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CargoRepository struct {
	db *gorm.DB
}

func NewCargoRepository() *CargoRepository {
	return &CargoRepository{db: configs.DB}
}

func (r *CargoRepository) WithContext(ctx context.Context) *CargoRepository {
	return &CargoRepository{db: r.db.WithContext(ctx)}
}

func (r *CargoRepository) FindAll() ([]models.Cargo, error) {
	var cargos []models.Cargo
	err := r.db.Order("codigo asc").Find(&cargos).Error
	return cargos, err
}

func (r *CargoRepository) Create(c *models.Cargo) error {
	return r.db.Create(c).Error
}

func (r *CargoRepository) FindByID(id string) (*models.Cargo, error) {
	var cargo models.Cargo
	err := r.db.First(&cargo, "id = ?", id).Error
	return &cargo, err
}

func (r *CargoRepository) Delete(id string) error {
	return r.db.Delete(&models.Cargo{}, "id = ?", id).Error
}

func (r *CargoRepository) FindByDescripcion(descripcion string) (*models.Cargo, error) {
	var cargo models.Cargo
	err := r.db.Where("descripcion ILIKE ?", descripcion).First(&cargo).Error
	return &cargo, err
}
func (r *CargoRepository) GetNextCodigo() (int, error) {
	var maxCodigo int
	err := r.db.Model(&models.Cargo{}).Select("COALESCE(MAX(codigo), 0)").Scan(&maxCodigo).Error
	return maxCodigo + 1, err
}
