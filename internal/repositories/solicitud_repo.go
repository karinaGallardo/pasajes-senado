package repositories

import (
	"context"
	"fmt"
	"sistema-pasajes/internal/models"
	"strings"
	"time"

	"gorm.io/gorm"
)

type SolicitudRepository struct {
	db *gorm.DB
}

type PaginatedSolicitudes struct {
	Solicitudes []models.Solicitud
	Total       int64
	Page        int
	Limit       int
	TotalPages  int
	SearchTerm  string
}

func NewSolicitudRepository(db *gorm.DB) *SolicitudRepository {
	return &SolicitudRepository{db: db}
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

func (r *SolicitudRepository) Create(ctx context.Context, solicitud *models.Solicitud) error {
	return r.db.WithContext(ctx).Create(solicitud).Error
}

func (r *SolicitudRepository) CreateWithSequenceCode(ctx context.Context, solicitud *models.Solicitud, prefix string, seqRepo *CodigoSecuenciaRepository) error {
	currentYear := time.Now().Year()
	for {
		nextVal, err := seqRepo.WithTx(r.db).GetNext(ctx, currentYear, prefix)
		if err != nil {
			return err
		}
		solicitud.Codigo = fmt.Sprintf("%s-%d%04d", prefix, currentYear%100, nextVal)

		// Verificar si ya existe (incluyendo eliminados) para evitar errores de clave duplicada
		exists, err := r.ExistsByCodigo(ctx, solicitud.Codigo)
		if err != nil {
			return err
		}
		if !exists {
			break
		}
	}

	return r.Create(ctx, solicitud)
}

func SearchSolicitud(term string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db
		}

		words := strings.Fields(term)
		for _, word := range words {
			likeTerm := "%" + word + "%"
			// Search in Solicitud Codigo or joined Usuario fields
			db = db.Where("(solicitudes.codigo ILIKE ? OR solicitudes.id::text ILIKE ? OR usuarios.firstname ILIKE ? OR usuarios.lastname ILIKE ? OR usuarios.ci ILIKE ? OR usuarios.username ILIKE ?)",
				likeTerm, likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
		}

		return db
	}
}

func (r *SolicitudRepository) FindAll(ctx context.Context, status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
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

func (r *SolicitudRepository) FindByUserID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
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

func (r *SolicitudRepository) FindByCupoDerechoItemID(ctx context.Context, itemID string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Usuario.Encargado").
		Preload("Usuario.Oficina").
		Preload("Usuario.Departamento").
		Preload("Usuario.Origen").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("TipoItinerario").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Preload("CupoDerechoItem").
		Where("cupo_derecho_item_id = ?", itemID).
		Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindByUserIdOrAccesibleByEncargadoID(ctx context.Context, userID string, status string, concepto string) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Order("created_at desc").
		Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.WithContext(ctx).Table("usuarios").Select("id").Where("encargado_id = ?", userID),
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

func (r *SolicitudRepository) FindPaginated(ctx context.Context, userID string, isAdmin bool, status string, concepto string, page, limit int, searchTerm string) (*PaginatedSolicitudes, error) {
	var solicitudes []models.Solicitud
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&models.Solicitud{}).
		Joins("LEFT JOIN usuarios ON solicitudes.usuario_id = usuarios.id").
		Preload("Usuario").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Estado").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Scopes(SearchSolicitud(searchTerm))

	if !isAdmin {
		baseQuery = baseQuery.Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.WithContext(ctx).Table("usuarios").Select("id").Where("encargado_id = ?", userID),
		)
	}

	if status != "" {
		baseQuery = baseQuery.Where("solicitudes.estado_solicitud_codigo = ?", status)
	}

	if concepto != "" {
		baseQuery = baseQuery.Joins("JOIN tipo_solicitudes ON solicitudes.tipo_solicitud_codigo = tipo_solicitudes.codigo").
			Where("tipo_solicitudes.concepto_viaje_codigo = ?", concepto)
	}

	baseQuery.Count(&total)

	err := baseQuery.
		Scopes(Paginate(page, limit)).
		Order("solicitudes.created_at DESC").
		Find(&solicitudes).Error

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedSolicitudes{
		Solicitudes: solicitudes,
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		SearchTerm:  searchTerm,
	}, err
}

func (r *SolicitudRepository) FindByID(ctx context.Context, id string) (*models.Solicitud, error) {
	var solicitud models.Solicitud
	err := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Usuario.Encargado").
		Preload("Usuario.OrigenesAlternativos.Destino").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen.Ambito").
		Preload("Items.Destino.Ambito").
		Preload("Items.Aerolinea").
		Preload("Items.Pasajes.Aerolinea").
		Preload("Items.Pasajes.Agencia").
		Preload("Items.Pasajes.EstadoPasaje").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("Viaticos").
		Preload("Viaticos.Detalles", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Preload("TipoItinerario").
		Preload("AmbitoViaje").
		Preload("CupoDerechoItem").
		First(&solicitud, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &solicitud, nil
}

func (r *SolicitudRepository) Update(ctx context.Context, solicitud *models.Solicitud) error {
	return r.db.WithContext(ctx).Session(&gorm.Session{FullSaveAssociations: true}).Save(solicitud).Error
}

func (r *SolicitudRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.Solicitud{}).Where("id = ?", id).Update("estado_solicitud_codigo", status).Error
}

func (r *SolicitudRepository) Delete(ctx context.Context, id string, deletedBy string) error {
	// First update DeletedBy, then call Delete (soft delete)
	if err := r.db.WithContext(ctx).Model(&models.Solicitud{}).Where("id = ?", id).Update("deleted_by", deletedBy).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Delete(&models.Solicitud{}, "id = ?", id).Error
}

func (r *SolicitudRepository) ExistsByCodigo(ctx context.Context, codigo string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Unscoped().Model(&models.Solicitud{}).Where("codigo = ?", codigo).Count(&count).Error
	return count > 0, err
}

func (r *SolicitudRepository) FindPendientesDeDescargo(ctx context.Context) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	err := r.db.WithContext(ctx).Preload("Usuario.Encargado").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("Descargo.Tramos").
		Preload("Descargo.Oficial").
		Joins("LEFT JOIN descargos ON solicitudes.id = descargos.solicitud_id").
		Where("(descargos.id IS NULL OR descargos.estado != ?)", models.EstadoDescargoAprobado).
		Where("solicitudes.estado_solicitud_codigo IN (?)", []string{"PARCIALMENTE_APROBADO", "APROBADO", "EMITIDO"}).
		Where("EXISTS (SELECT 1 FROM pasajes p JOIN solicitud_items si ON p.solicitud_item_id = si.id WHERE si.solicitud_id = solicitudes.id AND p.estado_pasaje_codigo = 'EMITIDO')").
		Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindPendientesDeDescargoUI(ctx context.Context, userID string, isAdmin bool) ([]models.Solicitud, error) {
	var solicitudes []models.Solicitud
	query := r.db.WithContext(ctx).Preload("Usuario").
		Preload("Usuario.Rol").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Preload("Descargo.Tramos").
		Preload("Descargo.Oficial").
		Joins("LEFT JOIN descargos ON solicitudes.id = descargos.solicitud_id").
		Where("(descargos.id IS NULL OR descargos.estado != ?)", models.EstadoDescargoAprobado).
		Where("solicitudes.estado_solicitud_codigo IN (?)", []string{"PARCIALMENTE_APROBADO", "APROBADO", "EMITIDO"}).
		Where("EXISTS (SELECT 1 FROM pasajes p JOIN solicitud_items si ON p.solicitud_item_id = si.id WHERE si.solicitud_id = solicitudes.id AND p.estado_pasaje_codigo = 'EMITIDO')").
		Order("created_at desc")

	if !isAdmin {
		query = query.Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.WithContext(ctx).Table("usuarios").Select("id").Where("encargado_id = ?", userID),
		)
	}

	err := query.Find(&solicitudes).Error
	return solicitudes, err
}

func (r *SolicitudRepository) FindPendientesDeDescargoPaginated(ctx context.Context, userID string, isAdmin bool, page, limit int, searchTerm string) (*PaginatedSolicitudes, error) {
	var solicitudes []models.Solicitud
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&models.Solicitud{}).
		Preload("Usuario").
		Preload("Usuario.Rol").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Origen").
		Preload("Items.Destino").
		Preload("Items.Pasajes.RutaPasaje").
		Preload("Items.Pasajes.RutaPasaje.Origen").
		Preload("Items.Pasajes.RutaPasaje.Destino").
		Preload("Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB {
			return db.Order("seq ASC")
		}).
		Preload("Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("TipoSolicitud.ConceptoViaje").
		Preload("EstadoSolicitud").
		Preload("Aerolinea").
		Preload("Descargo.Tramos").
		Preload("Descargo.Oficial").
		Joins("LEFT JOIN descargos ON solicitudes.id = descargos.solicitud_id").
		Joins("LEFT JOIN usuarios ON solicitudes.usuario_id = usuarios.id").
		Where("(descargos.id IS NULL OR descargos.estado != ?)", models.EstadoDescargoAprobado).
		Where("solicitudes.estado_solicitud_codigo IN (?)", []string{"PARCIALMENTE_APROBADO", "APROBADO", "EMITIDO"}).
		Where("EXISTS (SELECT 1 FROM pasajes p JOIN solicitud_items si ON p.solicitud_item_id = si.id WHERE si.solicitud_id = solicitudes.id AND p.estado_pasaje_codigo = 'EMITIDO')").
		Scopes(SearchSolicitud(searchTerm))

	if !isAdmin {
		baseQuery = baseQuery.Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.WithContext(ctx).Table("usuarios").Select("id").Where("encargado_id = ?", userID),
		)
	}

	baseQuery.Count(&total)

	err := baseQuery.
		Scopes(Paginate(page, limit)).
		Order("solicitudes.created_at DESC").
		Find(&solicitudes).Error

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedSolicitudes{
		Solicitudes: solicitudes,
		Total:       total,
		Page:        page,
		Limit:       limit,
		TotalPages:  totalPages,
		SearchTerm:  searchTerm,
	}, err
}
func (r *SolicitudRepository) CountPending(ctx context.Context, userID string, isAdmin bool) (int64, error) {
	var count int64
	baseQuery := r.db.WithContext(ctx).Model(&models.Solicitud{})

	if !isAdmin {
		baseQuery = baseQuery.Where(
			"solicitudes.usuario_id = ? OR solicitudes.created_by = ? OR solicitudes.usuario_id IN (?)",
			userID,
			userID,
			r.db.WithContext(ctx).Table("usuarios").Select("id").Where("encargado_id = ?", userID),
		)
	}

	err := baseQuery.Where("solicitudes.estado_solicitud_codigo = ?", "SOLICITADO").Count(&count).Error
	return count, err
}

func (r *SolicitudRepository) UpdateTimestamps(ctx context.Context, id string, createdAt, updatedAt any) error {
	return r.db.WithContext(ctx).Model(&models.Solicitud{}).Where("id = ?", id).Updates(map[string]interface{}{
		"created_at": createdAt,
		"updated_at": updatedAt,
	}).Error
}
