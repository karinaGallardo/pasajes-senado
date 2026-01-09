package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DestinoRepository struct {
	db *gorm.DB
}

func NewDestinoRepository() *DestinoRepository {
	return &DestinoRepository{db: configs.DB}
}

func (r *DestinoRepository) WithContext(ctx context.Context) *DestinoRepository {
	return &DestinoRepository{db: r.db.WithContext(ctx)}
}

func (r *DestinoRepository) FindAll() ([]models.Destino, error) {
	var list []models.Destino
	err := r.db.Preload("Ambito").Preload("Departamento").Order("ciudad asc").Find(&list).Error
	return list, err
}

func (r *DestinoRepository) FindByAmbito(ambitoCodigo string) ([]models.Destino, error) {
	var list []models.Destino
	err := r.db.Preload("Ambito").Preload("Departamento").Where("ambito_codigo = ?", ambitoCodigo).Order("ciudad asc").Find(&list).Error
	return list, err
}

func (r *DestinoRepository) FindByIATA(iata string) (*models.Destino, error) {
	var d models.Destino
	err := r.db.Preload("Ambito").Preload("Departamento").Where("iata = ?", iata).First(&d).Error
	return &d, err
}

func (r *DestinoRepository) Create(d *models.Destino) error {
	return r.db.Create(d).Error
}

func (r *DestinoRepository) Save(d *models.Destino) error {
	return r.db.Save(d).Error
}
