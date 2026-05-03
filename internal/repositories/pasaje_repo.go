package repositories

import (
	"context"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"

	"gorm.io/gorm"
)

type PasajeRepository struct {
	db *gorm.DB
}

func NewPasajeRepository(db *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: db}
}

func (r *PasajeRepository) Create(ctx context.Context, pasaje *models.Pasaje) error {
	return r.db.WithContext(ctx).Create(pasaje).Error
}

func (r *PasajeRepository) FindBySolicitudID(ctx context.Context, solicitudID string) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	err := r.db.WithContext(ctx).Where("solicitud_id = ?", solicitudID).Order("seq ASC").Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) Delete(ctx context.Context, id string, deletedBy string) error {
	if err := r.db.WithContext(ctx).Model(&models.Pasaje{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&models.Pasaje{}, "id = ?", id).Error
}

func (r *PasajeRepository) FindByID(ctx context.Context, id string) (*models.Pasaje, error) {
	var pasaje models.Pasaje
	err := r.db.WithContext(ctx).
		Preload("EstadoPasaje").
		Preload("Agencia").
		Preload("Aerolinea").
		Preload("Cargos", func(db *gorm.DB) *gorm.DB { return db.Order("created_at DESC") }).
		Preload("RutaPasaje.Origen").
		Preload("RutaPasaje.Destino").
		Preload("RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("RutaPasaje.Escalas.Destino").
		Preload("SolicitudItem.Origen.Ambito").
		Preload("SolicitudItem.Destino.Ambito").
		Preload("SolicitudItem.Solicitud.CupoDerechoItem").
		Preload("SolicitudItem.Solicitud.Usuario").
		Preload("DescargoTramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("DescargoTramos.Origen").
		Preload("DescargoTramos.Destino").
		First(&pasaje, "id = ?", id).Error
	return &pasaje, err
}

func (r *PasajeRepository) Update(ctx context.Context, pasaje *models.Pasaje) error {
	return r.db.WithContext(ctx).Save(pasaje).Error
}

func (r *PasajeRepository) WithTx(tx *gorm.DB) *PasajeRepository {
	return &PasajeRepository{db: tx}
}

func (r *PasajeRepository) WithContext(ctx context.Context) *PasajeRepository {
	return &PasajeRepository{db: r.db.WithContext(ctx)}
}

func (r *PasajeRepository) RunTransaction(fn func(repo *PasajeRepository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo, tx)
	})
}

func (r *PasajeRepository) FindByNumeroBillete(ctx context.Context, numeroBillete string) (*models.Pasaje, error) {
	var pasaje models.Pasaje
	err := r.db.WithContext(ctx).Where("numero_billete = ?", numeroBillete).First(&pasaje).Error
	return &pasaje, err
}

func (r *PasajeRepository) FindConsolidado(ctx context.Context, filter dtos.ReportFilterRequest) ([]models.Pasaje, error) {
	var pasajes []models.Pasaje
	query := r.db.WithContext(ctx).Model(&models.Pasaje{}).
		Preload("EstadoPasaje").
		Preload("Aerolinea").
		Preload("Agencia").
		Preload("RutaPasaje.Origen").
		Preload("RutaPasaje.Destino").
		Preload("RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("RutaPasaje.Escalas.Destino").
		Preload("SolicitudItem.Origen").
		Preload("SolicitudItem.Destino").
		Preload("SolicitudItem").
		Joins("INNER JOIN solicitudes ON solicitudes.id = pasajes.solicitud_id").
		Joins("INNER JOIN usuarios ON usuarios.id = solicitudes.usuario_id").
		Preload("SolicitudItem.Solicitud.Usuario").
		Preload("SolicitudItem.Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("SolicitudItem.Solicitud.Descargo").
		Order("pasajes.fecha_emision DESC")

	if filter.FechaDesde != "" {
		query = query.Where("pasajes.fecha_emision >= ?", filter.FechaDesde)
	}
	if filter.FechaHasta != "" {
		query = query.Where("pasajes.fecha_emision <= ?", filter.FechaHasta)
	}
	if filter.AerolineaID != "" {
		query = query.Where("pasajes.aerolinea_id = ?", filter.AerolineaID)
	}
	if filter.AgenciaID != "" {
		query = query.Where("pasajes.agencia_id = ?", filter.AgenciaID)
	}
	if filter.Estado != "" && filter.Estado != "ALL" {
		query = query.Where("pasajes.estado_pasaje_codigo = ?", filter.Estado)
	}

	if filter.Concepto != "" && filter.Concepto != "ALL" {
		// DERECHO or OFICIAL
		// We join with tipo_solicitudes to check ConceptoViaje
		query = query.Joins("INNER JOIN tipo_solicitudes ts ON ts.codigo = solicitudes.tipo_solicitud_codigo").
			Joins("INNER JOIN concepto_viajes cv ON cv.codigo = ts.concepto_viaje_codigo").
			Where("cv.codigo = ?", filter.Concepto)
	}

	err := query.Find(&pasajes).Error
	return pasajes, err
}

func (r *PasajeRepository) GetDB() *gorm.DB {
	return r.db
}
