package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type ConfiguracionRepository struct {
	db *gorm.DB
}

func NewConfiguracionRepository() *ConfiguracionRepository {
	return &ConfiguracionRepository{db: configs.DB}
}

func (r *ConfiguracionRepository) WithContext(ctx context.Context) *ConfiguracionRepository {
	return &ConfiguracionRepository{db: r.db.WithContext(ctx)}
}

func (r *ConfiguracionRepository) FindAll() ([]models.Configuracion, error) {
	var list []models.Configuracion
	err := r.db.Order("clave asc").Find(&list).Error
	return list, err
}

func (r *ConfiguracionRepository) FindByClave(clave string) (*models.Configuracion, error) {
	var conf models.Configuracion
	err := r.db.Where("clave = ?", clave).First(&conf).Error
	return &conf, err
}

func (r *ConfiguracionRepository) Update(conf *models.Configuracion) error {
	return r.db.Save(conf).Error
}
