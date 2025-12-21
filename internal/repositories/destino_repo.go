package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type DestinoRepository struct{}

func NewDestinoRepository() *DestinoRepository {
	return &DestinoRepository{}
}

func (r *DestinoRepository) FindAll() ([]models.Ciudad, error) {
	var destinos []models.Ciudad
	err := configs.DB.Order("ciudad asc").Find(&destinos).Error
	return destinos, err
}

func (r *DestinoRepository) Create(destino *models.Ciudad) error {
	return configs.DB.Create(destino).Error
}

func (r *DestinoRepository) SeedDefaults() error {
	var count int64
	configs.DB.Model(&models.Ciudad{}).Count(&count)
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

	return configs.DB.Create(&defaults).Error
}
