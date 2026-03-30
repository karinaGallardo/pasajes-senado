package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type ConfiguracionRepository struct {
	db *gorm.DB
}

func NewConfiguracionRepository(db *gorm.DB) *ConfiguracionRepository {
	return &ConfiguracionRepository{db: db}
}

func (r *ConfiguracionRepository) WithContext(ctx context.Context) *ConfiguracionRepository {
	return &ConfiguracionRepository{db: r.db.WithContext(ctx)}
}

func (r *ConfiguracionRepository) FindAll(ctx context.Context) ([]models.Configuracion, error) {
	var list []models.Configuracion
	err := r.db.WithContext(ctx).Order("clave asc").Find(&list).Error
	return list, err
}

func (r *ConfiguracionRepository) FindByClave(ctx context.Context, clave string) (*models.Configuracion, error) {
	var conf models.Configuracion
	err := r.db.WithContext(ctx).Where("clave = ?", clave).First(&conf).Error
	return &conf, err
}

func (r *ConfiguracionRepository) Update(ctx context.Context, conf *models.Configuracion) error {
	return r.db.WithContext(ctx).Save(conf).Error
}
