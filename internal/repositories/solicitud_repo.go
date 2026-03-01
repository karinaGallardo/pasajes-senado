package repositories

import (
	"context"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type SolicitudRepository struct {
	db *gorm.DB
}

func NewSolicitudRepository() *SolicitudRepository {
	return &SolicitudRepository{db: configs.DB}
}

func (r *SolicitudRepository) WithTx(tx *gorm.DB) *SolicitudRepository {
	return &SolicitudRepository{db: tx}
}

func (r *SolicitudRepository) WithContext(ctx context.Context) *SolicitudRepository {
	return &SolicitudRepository{db: r.db.WithContext(ctx)}
}

func (r *SolicitudRepository) RunTransaction(fn func(repo *SolicitudRepository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo, tx)
	})
}

func (r *SolicitudRepository) Create(solicitud *models.Solicitud) error {
	return r.db.Create(solicitud).Error
}

func (r *SolicitudRepository) FindAll(status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.Preload("Usuario").
		Preload("Items").
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Order("created_at desc")

	if status != "" {
		query = query.Where("estado_solicitud_codigo = ?", status)
	}

	if concepto != "" {
		query = query.Joins("JOIN tipo_solicitudes ON solicitudes.tipo_solicitud_codigo = tipo_solicitudes.codigo").
			Where("tipo_solicitudes.concepto_viaje_codigo = ?", concepto)
	}

	err := query.Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByUserID(userID string, status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.Preload("Usuario").
		Preload("Items").
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Order("created_at desc").Where("usuario_id = ?", userID)

	if status != "" {
		query = query.Where("estado_solicitud_codigo = ?", status)
	}

	if concepto != "" {
		query = query.Joins("JOIN tipo_solicitudes ON solicitudes.tipo_solicitud_codigo = tipo_solicitudes.codigo").
			Where("tipo_solicitudes.concepto_viaje_codigo = ?", concepto)
	}

	err := query.Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByCupoDerechoItemID(itemID string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario").
		Preload("Usuario.Encargado").
		Preload("Usuario.Oficina").
		Preload("Usuario.Departamento").
		Preload("Usuario.Origen").
		Preload("Items").
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("TipoItinerario").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("CupoDerechoItem").
		Where("cupo_derecho_item_id = ?", itemID).
		Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByUserIdOrAccesibleByEncargadoID(userID string, status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.Preload("Usuario").
		Preload("Items").
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Order("created_at desc").
		Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.Table("usuarios").Select("id").Where("encargado_id = ?", userID),
		)

	if status != "" {
		query = query.Where("estado_solicitud_codigo = ?", status)
	}

	if concepto != "" {
		query = query.Joins("JOIN tipo_solicitudes ON solicitudes.tipo_solicitud_codigo = tipo_solicitudes.codigo").
			Where("tipo_solicitudes.concepto_viaje_codigo = ?", concepto)
	}

	err := query.Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByID(id string) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := r.db.Preload("Usuario").
		Preload("Usuario.Encargado").
		Preload("Items").
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Destino").
		Preload("Items.Pasajes.Aerolinea").
		Preload("Items.Pasajes.Agencia").
		Preload("Items.Pasajes.EstadoPasaje").
		Preload("Items.Pasajes").
		Preload("Viaticos").
		Preload("Viaticos.Detalles").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("TipoItinerario").
		Preload("AmbitoViaje").
		Preload("CupoDerechoItem").
		First(&solicitud, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &solicitud, nil
}

func (r *SolicitudRepository) Update(solicitud *models.Solicitud) error {
	return r.db.Save(solicitud).Error
}

func (r *SolicitudRepository) UpdateStatus(id string, status string) error {
	return r.db.Model(&models.Solicitud{}).Where("id = ?", id).Update("estado_solicitud_codigo", status).Error
}

func (r *SolicitudRepository) Delete(id string) error {
	return r.db.Delete(&models.Solicitud{}, "id = ?", id).Error
}

func (r *SolicitudRepository) ExistsByCodigo(codigo string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Solicitud{}).Where("codigo = ?", codigo).Count(&count).Error
	return count > 0, err
}

func (r *SolicitudRepository) FindPendientesDeDescargo() ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario.Encargado").
		Preload("Items").
		Joins("LEFT JOIN descargos ON solicitudes.id = descargos.solicitud_id").
		Where("descargos.id IS NULL").
		Where("solicitudes.estado_solicitud_codigo IN (?)", []string{"EMITIDO", "FINALIZADO"}).
		Find(&solicitudes).Error
	return solicitudes, err
}
