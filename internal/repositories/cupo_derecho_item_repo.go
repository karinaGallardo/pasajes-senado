package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CupoDerechoItemRepository struct {
	db *gorm.DB
}

func NewCupoDerechoItemRepository(db *gorm.DB) *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: db}
}

func (r *CupoDerechoItemRepository) WithTx(tx *gorm.DB) *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: tx}
}

func (r *CupoDerechoItemRepository) WithContext(ctx context.Context) *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: r.db.WithContext(ctx)}
}

func (r *CupoDerechoItemRepository) CreateInBatches(ctx context.Context, items []models.CupoDerechoItem, batchSize int) error {
	return r.db.WithContext(ctx).CreateInBatches(items, batchSize).Error
}

func (r *CupoDerechoItemRepository) Create(ctx context.Context, item *models.CupoDerechoItem) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *CupoDerechoItemRepository) FindByHolderAndPeriodo(ctx context.Context, userID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_asignado_id = ? AND gestion = ? AND mes = ?", userID, gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindByHolderAndGestion(ctx context.Context, userID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_asignado_id = ? AND gestion = ?", userID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindAvailableByHolderAndPeriodo(ctx context.Context, userID string, gestion, mes int) (*models.CupoDerechoItem, error) {
	var item models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Where("sen_asignado_id = ? AND gestion = ? AND mes = ? AND estado_cupo_derecho_codigo = 'DISPONIBLE'", userID, gestion, mes).
		First(&item).Error
	return &item, err
}

func (r *CupoDerechoItemRepository) FindForTitularByPeriodo(ctx context.Context, senadorID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("SenAsignado").
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_titular_id = ? AND gestion = ? AND mes = ?", senadorID, gestion, mes).
		Order("semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForSuplenteByPeriodo(ctx context.Context, beneficiarioID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_asignado_id = ? AND es_transferido = true AND gestion = ? AND mes = ?", beneficiarioID, gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForTitularByGestion(ctx context.Context, senadorID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("SenAsignado").
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_titular_id = ? AND gestion = ?", senadorID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForSuplenteByGestion(ctx context.Context, beneficiarioID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Preload("Solicitudes.Items").
		Where("sen_asignado_id = ? AND es_transferido = true AND gestion = ?", beneficiarioID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) Update(ctx context.Context, item *models.CupoDerechoItem) error {
	return r.db.WithContext(ctx).Omit(clause.Associations).Save(item).Error
}

func (r *CupoDerechoItemRepository) FindByPeriodo(ctx context.Context, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("SenTitular").
		Preload("SenAsignado").
		Where("gestion = ? AND mes = ?", gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindByID(ctx context.Context, id string) (*models.CupoDerechoItem, error) {
	var v models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("SenTitular").
		Preload("SenAsignado").
		Preload("Solicitudes.Items").
		First(&v, "id = ?", id).
		Error
	return &v, err
}

func (r *CupoDerechoItemRepository) DeleteUnscoped(ctx context.Context, v *models.CupoDerechoItem) error {
	return r.db.WithContext(ctx).Unscoped().Delete(v).Error
}

func (r *CupoDerechoItemRepository) FindByCupoDerechoID(ctx context.Context, cupoDerechoID string) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.WithContext(ctx).
		Preload("SenTitular").
		Preload("SenAsignado").
		Where("cupo_derecho_id = ?", cupoDerechoID).
		Find(&list).
		Error
	return list, err
}

func (r *CupoDerechoItemRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.CupoDerechoItem{}).Where("id = ?", id).Update("estado_cupo_derecho_codigo", status).Error
}
