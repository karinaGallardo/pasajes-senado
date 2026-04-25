package repositories

import (
	"context"
	"sistema-pasajes/internal/models"
	"strings"

	"gorm.io/gorm"
)

type DestinoRepository struct {
	db *gorm.DB
}

func NewDestinoRepository(db *gorm.DB) *DestinoRepository {
	return &DestinoRepository{db: db}
}

func (r *DestinoRepository) WithContext(ctx context.Context) *DestinoRepository {
	return &DestinoRepository{db: r.db.WithContext(ctx)}
}

func (r *DestinoRepository) FindAll(ctx context.Context) ([]models.Destino, error) {
	var list []models.Destino
	err := r.db.WithContext(ctx).Preload("Ambito").Preload("Departamento").Order("ciudad asc").Find(&list).Error
	return list, err
}

func (r *DestinoRepository) FindByAmbito(ctx context.Context, ambitoCodigo string) ([]models.Destino, error) {
	var list []models.Destino
	err := r.db.WithContext(ctx).Preload("Ambito").Preload("Departamento").Where("ambito_codigo = ?", ambitoCodigo).Order("ciudad asc").Find(&list).Error
	return list, err
}

func (r *DestinoRepository) FindByIATA(ctx context.Context, iata string) (*models.Destino, error) {
	var d models.Destino
	err := r.db.WithContext(ctx).Preload("Ambito").Preload("Departamento").Where("iata = ?", iata).First(&d).Error
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DestinoRepository) Create(ctx context.Context, d *models.Destino) error {
	return r.db.WithContext(ctx).Create(d).Error
}

func (r *DestinoRepository) Update(ctx context.Context, d *models.Destino) error {
	return r.db.WithContext(ctx).Save(d).Error
}

func (r *DestinoRepository) Search(ctx context.Context, query string, ambito string, page, pageSize int) ([]models.Destino, int64, error) {
	var list []models.Destino
	var total int64
	words := strings.Fields(query)

	db := r.db.WithContext(ctx).Model(&models.Destino{})

	if ambito != "" && ambito != "INTERNACIONAL" {
		db = db.Where("ambito_codigo = ?", ambito)
	}

	for _, word := range words {
		q := "%" + word + "%"
		db = db.Where("(iata ILIKE ? OR ciudad ILIKE ? OR aeropuerto ILIKE ?)", q, q, q)
	}

	// Count total records before pagination
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination and preloads
	offset := (page - 1) * pageSize
	err := db.Preload("Ambito").
		Preload("Departamento").
		Order("ciudad asc").
		Limit(pageSize).
		Offset(offset).
		Find(&list).Error

	return list, total, err
}

func (r *DestinoRepository) Delete(ctx context.Context, iata string) error {
	return r.db.WithContext(ctx).Where("iata = ?", iata).Delete(&models.Destino{}).Error
}
