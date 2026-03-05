package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type CargoRepository struct {
	db *gorm.DB
}

func NewCargoRepository(db *gorm.DB) *CargoRepository {
	return &CargoRepository{db: db}
}

func (r *CargoRepository) WithContext(ctx context.Context) *CargoRepository {
	return &CargoRepository{db: r.db.WithContext(ctx)}
}

func (r *CargoRepository) FindAll(ctx context.Context) ([]models.Cargo, error) {
	var cargos []models.Cargo
	err := r.db.WithContext(ctx).Order("codigo asc").Find(&cargos).Error
	return cargos, err
}

func (r *CargoRepository) Create(ctx context.Context, c *models.Cargo) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *CargoRepository) FindByID(ctx context.Context, id string) (*models.Cargo, error) {
	var cargo models.Cargo
	err := r.db.WithContext(ctx).First(&cargo, "id = ?", id).Error
	return &cargo, err
}

func (r *CargoRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Cargo{}, "id = ?", id).Error
}

func (r *CargoRepository) FindByDescripcion(ctx context.Context, descripcion string) (*models.Cargo, error) {
	var cargo models.Cargo
	err := r.db.WithContext(ctx).Where("descripcion ILIKE ?", descripcion).First(&cargo).Error
	return &cargo, err
}
func (r *CargoRepository) GetNextCodigo(ctx context.Context) (int, error) {
	var maxCodigo int
	err := r.db.WithContext(ctx).Model(&models.Cargo{}).Select("COALESCE(MAX(codigo), 0)").Scan(&maxCodigo).Error
	return maxCodigo + 1, err
}
