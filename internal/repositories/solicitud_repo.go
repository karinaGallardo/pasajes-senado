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

func (r *SolicitudRepository) FindAll() ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("TipoSolicitud.ConceptoViaje").Preload("EstadoSolicitud").Order("created_at desc").Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByUserID(userID string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("TipoSolicitud.ConceptoViaje").Preload("EstadoSolicitud").Order("created_at desc").Where("usuario_id = ?", userID).Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByUserIdOrAccesibleByEncargadoID(userID string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("TipoSolicitud.ConceptoViaje").Preload("EstadoSolicitud").
		Order("created_at desc").
		Where("usuario_id = ? OR usuario_id IN (?)", userID, r.db.Table("usuarios").Select("id").Where("encargado_id = ?", userID)).
		Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByID(id string) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("Pasajes.Aerolinea").Preload("Pasajes.Agencia").Preload("Pasajes.EstadoPasaje").Preload("Pasajes").Preload("Viaticos").Preload("TipoSolicitud.ConceptoViaje").Preload("EstadoSolicitud").Preload("TipoItinerario").Preload("AmbitoViaje").First(&solicitud, "id = ?", id).Error
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
