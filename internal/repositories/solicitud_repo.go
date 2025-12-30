package repositories

import (
	"sistema-pasajes/internal/models"
	"time"

	"sistema-pasajes/internal/configs"
	"gorm.io/gorm"
)

type SolicitudRepository struct {
	db *gorm.DB
}

func NewSolicitudRepository() *SolicitudRepository {
	return &SolicitudRepository{db: configs.DB}
}

func (r *SolicitudRepository) Create(solicitud *models.Solicitud) error {
	return r.db.Create(solicitud).Error
}

func (r *SolicitudRepository) FindAll() ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("TipoSolicitud.ConceptoViaje").Order("created_at desc").Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByID(id string) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := r.db.Preload("Usuario").Preload("Origen").Preload("Destino").Preload("Pasajes").Preload("Viaticos").Preload("TipoSolicitud.ConceptoViaje").First(&solicitud, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &solicitud, nil
}

func (r *SolicitudRepository) Update(solicitud *models.Solicitud) error {
	return r.db.Save(solicitud).Error
}

func (r *SolicitudRepository) ExistsByCodigo(codigo string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Solicitud{}).Where("codigo = ?", codigo).Count(&count).Error
	return count > 0, err
}

func (r *SolicitudRepository) CountApprovedByConcepto(usuarioID string, year int, month int, conceptoCodigo string) (int64, error) {
	var count int64
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	err := r.db.Model(&models.Solicitud{}).
		Joins("JOIN tipo_solicituds ts ON solicitudes.tipo_solicitud_id = ts.id").
		Joins("JOIN concepto_viajes cv ON ts.concepto_viaje_id = cv.id").
		Where("solicitudes.usuario_id = ? AND solicitudes.estado = 'APROBADO'", usuarioID).
		Where("solicitudes.fecha_salida >= ? AND solicitudes.fecha_salida < ?", startDate, endDate).
		Where("cv.codigo = ?", conceptoCodigo).
		Count(&count).Error

	return count, err
}
