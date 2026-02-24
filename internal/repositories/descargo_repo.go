package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type DescargoRepository struct {
	db *gorm.DB
}

func NewDescargoRepository() *DescargoRepository {
	return &DescargoRepository{db: configs.DB}
}

func (r *DescargoRepository) WithContext(ctx context.Context) *DescargoRepository {
	return &DescargoRepository{db: r.db.WithContext(ctx)}
}

func (r *DescargoRepository) Create(descargo *models.Descargo) error {
	return r.db.Create(descargo).Error
}

func (r *DescargoRepository) FindBySolicitudID(solicitudID string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.Preload("Documentos").
		Preload("DetallesItinerario").
		Preload("Solicitud").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		Where("solicitud_id = ?", solicitudID).First(&descargo).Error
	return &descargo, err
}

func (r *DescargoRepository) FindByID(id string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.Preload("Documentos").
		Preload("DetallesItinerario").
		Preload("Solicitud").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("Solicitud.Viaticos").
		Preload("Solicitud.Viaticos.Detalles").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		First(&descargo, "id = ?", id).Error
	return &descargo, err
}

func (r *DescargoRepository) FindAll() ([]models.Descargo, error) {
	var descargos []models.Descargo
	err := r.db.Preload("Solicitud").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Items").
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes").
		Preload("DetallesItinerario").
		Preload("Oficial").
		Preload("Oficial.Anexos").
		Order("created_at desc").Find(&descargos).Error
	return descargos, err
}
func (r *DescargoRepository) Update(descargo *models.Descargo) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(descargo).Error
}

func (r *DescargoRepository) UpdateOficial(oficial *models.DescargoOficial) error {
	return r.db.Save(oficial).Error
}

func (r *DescargoRepository) ClearDetalles(descargoID string) error {
	return r.db.Where("descargo_id = ?", descargoID).Delete(&models.DetalleItinerarioDescargo{}).Error
}

func (r *DescargoRepository) ClearAnexos(oficialID string) error {
	return r.db.Where("descargo_oficial_id = ?", oficialID).Delete(&models.AnexoDescargo{}).Error
}
