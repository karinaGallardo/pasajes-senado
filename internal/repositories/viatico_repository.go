package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type ViaticoRepository struct {
	db *gorm.DB
}

func NewViaticoRepository(db *gorm.DB) *ViaticoRepository {
	return &ViaticoRepository{db: db}
}

func (r *ViaticoRepository) WithTx(tx *gorm.DB) *ViaticoRepository {
	return &ViaticoRepository{db: tx}
}

func (r *ViaticoRepository) WithContext(ctx context.Context) *ViaticoRepository {
	return &ViaticoRepository{db: r.db.WithContext(ctx)}
}

func (r *ViaticoRepository) Create(ctx context.Context, viatico *models.Viatico) error {
	return r.db.WithContext(ctx).Create(viatico).Error
}

func (r *ViaticoRepository) Update(ctx context.Context, viatico *models.Viatico) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(viatico).Error
}

func (r *ViaticoRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Viatico{}, "id = ?", id).Error
}

func (r *ViaticoRepository) FindByID(ctx context.Context, id string) (*models.Viatico, error) {
	var viatico models.Viatico
	err := r.db.WithContext(ctx).
		Preload("Detalles").
		Preload("Usuario").
		Preload("Usuario.Rol").
		Preload("Solicitud").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		First(&viatico, "id = ?", id).Error
	return &viatico, err
}

func (r *ViaticoRepository) FindBySolicitudID(ctx context.Context, solicitudID string) ([]models.Viatico, error) {
	var viaticos []models.Viatico
	err := r.db.WithContext(ctx).Preload("Detalles").Where("solicitud_id = ?", solicitudID).Find(&viaticos).Error
	return viaticos, err
}

func (r *ViaticoRepository) FindAll(ctx context.Context) ([]models.Viatico, error) {
	var list []models.Viatico
	err := r.db.WithContext(ctx).Preload("Usuario").Preload("Solicitud").Order("fecha_asignacion desc").Find(&list).Error
	return list, err
}
