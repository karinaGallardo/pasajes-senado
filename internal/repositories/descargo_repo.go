package repositories

import (
	"context"
	"sistema-pasajes/internal/models"
	"strings"

	"gorm.io/gorm"
)

type DescargoRepository struct {
	db *gorm.DB
}

type PaginatedDescargos struct {
	Descargos  []models.Descargo
	Total      int64
	Page       int
	Limit      int
	TotalPages int
	SearchTerm string
}

func NewDescargoRepository(db *gorm.DB) *DescargoRepository {
	return &DescargoRepository{db: db}
}

func (r *DescargoRepository) WithContext(ctx context.Context) *DescargoRepository {
	return &DescargoRepository{db: r.db.WithContext(ctx)}
}

func (r *DescargoRepository) WithTx(tx *gorm.DB) *DescargoRepository {
	return &DescargoRepository{db: tx}
}

func (r *DescargoRepository) RunTransaction(fn func(repo *DescargoRepository, tx *gorm.DB) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := r.WithTx(tx)
		return fn(txRepo, tx)
	})
}

func (r *DescargoRepository) Create(ctx context.Context, descargo *models.Descargo) error {
	return r.db.WithContext(ctx).Create(descargo).Error
}

func SearchDescargo(term string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db
		}

		words := strings.Fields(term)
		for _, word := range words {
			likeTerm := "%" + word + "%"
			db = db.Where("(descargos.codigo ILIKE ? OR usuarios.firstname ILIKE ? OR usuarios.lastname ILIKE ? OR usuarios.ci ILIKE ? OR solicitudes.codigo ILIKE ?)",
				likeTerm, likeTerm, likeTerm, likeTerm, likeTerm)
		}

		return db
	}
}

func (r *DescargoRepository) FindBySolicitudID(ctx context.Context, solicitudID string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.WithContext(ctx).
		Preload("Tramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje").
		Preload("Tramos.RutaPasaje.Origen").
		Preload("Tramos.RutaPasaje.Destino").
		Preload("Tramos.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.Origen").
		Preload("Tramos.Destino").
		Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.DescargoTramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Origen").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Origen").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Oficial").
		Preload("Oficial.Anexos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Oficial.TransportesTerrestres", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Where("solicitud_id = ?", solicitudID).First(&descargo).Error
	return &descargo, err
}

func (r *DescargoRepository) FindByID(ctx context.Context, id string) (*models.Descargo, error) {
	var descargo models.Descargo
	err := r.db.WithContext(ctx).
		Preload("Tramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje").
		Preload("Tramos.RutaPasaje.Origen").
		Preload("Tramos.RutaPasaje.Destino").
		Preload("Tramos.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.Origen").
		Preload("Tramos.Destino").
		Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.Usuario.Origen").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.DescargoTramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Origen").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Solicitud.Viaticos").
		Preload("Solicitud.Viaticos.Detalles").
		Preload("Oficial").
		Preload("Oficial.Anexos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Oficial.TransportesTerrestres", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		First(&descargo, "id = ?", id).Error
	return &descargo, err
}

func (r *DescargoRepository) FindAll(ctx context.Context) ([]models.Descargo, error) {
	var descargos []models.Descargo
	err := r.db.WithContext(ctx).Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.DescargoTramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Origen").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Origen").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Tramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje").
		Preload("Tramos.RutaPasaje.Origen").
		Preload("Tramos.RutaPasaje.Destino").
		Preload("Tramos.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje.Escalas.Destino").
		Preload("Tramos.Origen").
		Preload("Tramos.Destino").
		Preload("Oficial").
		Preload("Oficial.Anexos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Oficial.TransportesTerrestres", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Order("created_at desc").Find(&descargos).Error
	return descargos, err
}

func (r *DescargoRepository) FindCountByUserIDs(ctx context.Context, userIDs []string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Descargo{}).
		Joins("LEFT JOIN solicitudes ON descargos.solicitud_id = solicitudes.id").
		Where("solicitudes.usuario_id IN ?", userIDs).
		Count(&count).Error
	return count, err
}

func (r *DescargoRepository) FindPaginated(ctx context.Context, page, limit int, searchTerm string, userIDs []string) (*PaginatedDescargos, error) {
	var descargos []models.Descargo
	var total int64

	baseQuery := r.db.WithContext(ctx).Model(&models.Descargo{}).
		Joins("LEFT JOIN solicitudes ON descargos.solicitud_id = solicitudes.id").
		Joins("LEFT JOIN usuarios ON solicitudes.usuario_id = usuarios.id").
		Preload("Solicitud").
		Preload("Solicitud.TipoSolicitud.ConceptoViaje").
		Preload("Solicitud.EstadoSolicitud").
		Preload("Solicitud.Aerolinea").
		Preload("Solicitud.Usuario").
		Preload("Solicitud.Usuario.Oficina").
		Preload("Solicitud.Usuario.Cargo").
		Preload("Solicitud.Usuario.Origen").
		Preload("Solicitud.Usuario.Encargado").
		Preload("Solicitud.CupoDerechoItem").
		Preload("Solicitud.Items", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.DescargoTramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Origen").
		Preload("Solicitud.Items.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Origen").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Destino").
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Solicitud.Items.Pasajes.RutaPasaje.Escalas.Destino").
		Preload("Tramos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje").
		Preload("Tramos.RutaPasaje.Origen").
		Preload("Tramos.RutaPasaje.Destino").
		Preload("Tramos.RutaPasaje.Escalas", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Tramos.RutaPasaje.Escalas.Destino").
		Preload("Tramos.Origen").
		Preload("Tramos.Destino").
		Preload("Oficial").
		Preload("Oficial.Anexos", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Preload("Oficial.TransportesTerrestres", func(db *gorm.DB) *gorm.DB { return db.Order("seq ASC") }).
		Scopes(SearchDescargo(searchTerm))

	// Scope by user IDs when provided (non-admin)
	if len(userIDs) > 0 {
		baseQuery = baseQuery.Where("solicitudes.usuario_id IN ?", userIDs)
	}

	baseQuery.Count(&total)

	err := baseQuery.
		Scopes(Paginate(page, limit)).
		Order("descargos.created_at DESC").
		Find(&descargos).Error

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &PaginatedDescargos{
		Descargos:  descargos,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		SearchTerm: searchTerm,
	}, err
}

func (r *DescargoRepository) Update(ctx context.Context, descargo *models.Descargo) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Cargar el Descargo actual para comparar (Dirty Tracking)
		var existing models.Descargo
		if err := tx.First(&existing, "id = ?", descargo.ID).Error; err != nil {
			return err
		}

		if descargo.HasChanges(existing) {
			if err := tx.Model(descargo).Select("*").
				Omit("Tramos", "Oficial", "Anexos", "Terrestres", "CreatedAt", "CreatedBy").
				Updates(descargo).Error; err != nil {
				return err
			}
		}

		var newIDs []string
		for _, det := range descargo.Tramos {
			if det.ID != "" {
				newIDs = append(newIDs, det.ID)
			}
		}

		query := tx.Where("descargo_id = ?", descargo.ID)
		if len(newIDs) > 0 {
			query = query.Where("id NOT IN ?", newIDs)
		}
		if err := query.Delete(&models.DescargoTramo{}).Error; err != nil {
			return err
		}

		var existingDetails []models.DescargoTramo
		if err := tx.Where("descargo_id = ?", descargo.ID).Find(&existingDetails).Error; err != nil {
			return err
		}

		existingMap := make(map[string]models.DescargoTramo)
		for _, ed := range existingDetails {
			existingMap[ed.ID] = ed
		}

		for i := range descargo.Tramos {
			det := &descargo.Tramos[i]
			det.DescargoID = descargo.ID

			if det.ID == "" || strings.HasPrefix(det.ID, "new_") {
				det.ID = ""
				if err := tx.Create(det).Error; err != nil {
					return err
				}
			} else {
				if existing, ok := existingMap[det.ID]; ok {
					if det.HasChanges(existing) {
						if err := tx.Model(det).Select("*").Omit("CreatedAt", "CreatedBy").Updates(det).Error; err != nil {
							return err
						}
					}
				}
			}
		}

		return nil
	})
}

func (r *DescargoRepository) UpdateOficial(ctx context.Context, oficial *models.DescargoOficial) error {
	// 1. Carga el registro actual para comparar (Dirty Tracking)
	var existing models.DescargoOficial
	if err := r.db.WithContext(ctx).First(&existing, "id = ?", oficial.ID).Error; err != nil {
		// Si no existe (raro en un update pero posible), procedemos con un Save normal
		return r.db.WithContext(ctx).Save(oficial).Error
	}

	// 2. Solo actualizamos si cambió algo
	if oficial.HasChanges(existing) {
		return r.db.WithContext(ctx).Model(oficial).Select("*").Omit("CreatedAt", "CreatedBy").Updates(oficial).Error
	}

	return nil
}

func (r *DescargoRepository) FindOficialByDescargoID(ctx context.Context, descargoID string) (*models.DescargoOficial, error) {
	var oficial models.DescargoOficial
	err := r.db.WithContext(ctx).First(&oficial, "descargo_id = ?", descargoID).Error
	return &oficial, err
}

func (r *DescargoRepository) DeleteDetallesNotIn(ctx context.Context, descargoID string, ids []string) error {
	query := r.db.WithContext(ctx).Where("descargo_id = ?", descargoID)
	if len(ids) > 0 {
		query = query.Where("id NOT IN ?", ids)
	}
	return query.Delete(&models.DescargoTramo{}).Error
}

func (r *DescargoRepository) ClearAnexos(ctx context.Context, oficialID string) error {
	return r.db.WithContext(ctx).Where("descargo_oficial_id = ?", oficialID).Delete(&models.AnexoDescargo{}).Error
}

func (r *DescargoRepository) ClearTransportesTerrestres(ctx context.Context, oficialID string) error {
	return r.db.WithContext(ctx).Where("descargo_oficial_id = ?", oficialID).Delete(&models.TransporteTerrestreDescargo{}).Error
}
func (r *DescargoRepository) SaveAnexos(ctx context.Context, anexos []models.AnexoDescargo) error {
	if len(anexos) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&anexos).Error
}

func (r *DescargoRepository) SaveTransportesTerrestres(ctx context.Context, terrestres []models.TransporteTerrestreDescargo) error {
	if len(terrestres) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&terrestres).Error
}

func (r *DescargoRepository) GetDB() *gorm.DB {
	return r.db
}
