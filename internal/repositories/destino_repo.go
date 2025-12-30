package repositories

import (
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DestinoRepository struct {
	db *gorm.DB
}

func NewDestinoRepository(db *gorm.DB) *DestinoRepository {
	return &DestinoRepository{db: db}
}

func (r *DestinoRepository) FindAll() ([]models.Ciudad, error) {
	var destinos []models.Ciudad
	err := r.db.Order("ciudad asc").Find(&destinos).Error
	return destinos, err
}

func (r *DestinoRepository) Create(destino *models.Ciudad) error {
	return r.db.Create(destino).Error
}

func (r *DestinoRepository) SeedDefaults() error {
	var count int64
	r.db.Model(&models.Ciudad{}).Count(&count)
	if count > 0 {
		return nil
	}

	defaults := []models.Ciudad{
		{Nombre: "La Paz", Code: "LPZ"},
		{Nombre: "Santa Cruz", Code: "SCZ"},
		{Nombre: "Cochabamba", Code: "CBBA"},
		{Nombre: "Sucre", Code: "SUC"},
		{Nombre: "Tarija", Code: "TJA"},
		{Nombre: "Trinidad", Code: "TDD"},
		{Nombre: "Cobija", Code: "CJA"},
		{Nombre: "Oruro", Code: "ORU"},
		{Nombre: "Potosí", Code: "POT"},
		{Nombre: "Uyuni", Code: "UYU"},
	}

	return r.db.Create(&defaults).Error
}
