package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type DescargoRepository struct {
	db *gorm.DB
}

func NewDescargoRepository(db *gorm.DB) *DescargoRepository {
	return &DescargoRepository{db: db}
}

func (r *DescargoRepository) WithContext(ctx context.Context) *DescargoRepository {
	return &DescargoRepository{db: r.db.WithContext(ctx)}
}

func (r *DescargoRepository) Create(ctx context.Context, descargo *models.Descargo) error {
	return r.db.WithContext(ctx).Create(descargo).Error
}

func (r *DescargoRepository) FindBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.WithContext(ctx).Preload("Documentos").
		Preload("DetallesItinerario").
		Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		Where("solicitud_id = ?", solicitudID).First(&descargo).Error
	return &descargo, err
}

func (r *DescargoRepository) FindByID(ctx context.Context, id string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.WithContext(ctx).Preload("Documentos").
		Preload("DetallesItinerario").
		Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Solicitud.Viaticos").
		Preload("Solicitud.Viaticos.Detalles").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		First(&descargo, "id = ?", id).Error
	return &descargo, err
}

func (r *DescargoRepository) FindAll(ctx context.Context) ([]models.Descargo, error) {
	var descargos []models.Descargo
	err := r.db.WithContext(ctx).Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("Solicitud.Usuario.Encargado").
		Preload("DetallesItinerario").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		Order("created_at desc").Find(&descargos).Error
	return descargos, err
}
func (r *DescargoRepository) Update(ctx context.Context, descargo *models.Descargo) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(descargo).Error
}

func (r *DescargoRepository) UpdateOficial(ctx context.Context, oficial *models.DescargoOficial) error {
	return r.db.WithContext(ctx).Save(oficial).Error
}

func (r *DescargoRepository) ClearDetalles(ctx context.Context, descargoID string) error {
	return r.db.WithContext(ctx).Where("descargo_id = ?", descargoID).Delete(&models.DetalleItinerarioDescargo{}).Error
}

func (r *DescargoRepository) ClearAnexos(ctx context.Context, oficialID string) error {
	return r.db.WithContext(ctx).Where("descargo_oficial_id = ?", oficialID).Delete(&models.AnexoDescargo{}).Error
}
