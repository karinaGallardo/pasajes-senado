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
