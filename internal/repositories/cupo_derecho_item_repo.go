package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CupoDerechoItemRepository struct {
	db *gorm.DB
}

func NewCupoDerechoItemRepository() *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: configs.DB}
}

func (r *CupoDerechoItemRepository) WithTx(tx *gorm.DB) *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: tx}
}

func (r *CupoDerechoItemRepository) WithContext(ctx context.Context) *CupoDerechoItemRepository {
	return &CupoDerechoItemRepository{db: r.db.WithContext(ctx)}
}

func (r *CupoDerechoItemRepository) CreateInBatches(items []models.CupoDerechoItem, batchSize int) error {
	return r.db.CreateInBatches(items, batchSize).Error
}

func (r *CupoDerechoItemRepository) Create(item *models.CupoDerechoItem) error {
	return r.db.Create(item).Error
}

func (r *CupoDerechoItemRepository) FindByHolderAndPeriodo(userID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_asignado_id = ? AND gestion = ? AND mes = ?", userID, gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindByHolderAndGestion(userID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_asignado_id = ? AND gestion = ?", userID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindAvailableByHolderAndPeriodo(userID string, gestion, mes int) (*models.CupoDerechoItem, error) {
	var item models.CupoDerechoItem
	err := r.db.
		Where("sen_asignado_id = ? AND gestion = ? AND mes = ? AND estado_cupo_derecho_codigo = 'DISPONIBLE'", userID, gestion, mes).
		First(&item).Error
	return &item, err
}

func (r *CupoDerechoItemRepository) FindForTitularByPeriodo(senadorID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("SenAsignado").
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_titular_id = ? AND gestion = ? AND mes = ?", senadorID, gestion, mes).
		Order("semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForSuplenteByPeriodo(beneficiarioID string, gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_asignado_id = ? AND es_transferido = true AND gestion = ? AND mes = ?", beneficiarioID, gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForTitularByGestion(senadorID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("SenAsignado").
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_titular_id = ? AND gestion = ?", senadorID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindForSuplenteByGestion(beneficiarioID string, gestion int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("Solicitudes.Descargo").
		Preload("Solicitudes.TipoItinerario").
		Where("sen_asignado_id = ? AND es_transferido = true AND gestion = ?", beneficiarioID, gestion).
		Order("mes asc, semana asc").
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) Update(item *models.CupoDerechoItem) error {
	return r.db.Omit(clause.Associations).Save(item).Error
}

func (r *CupoDerechoItemRepository) FindByPeriodo(gestion, mes int) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("SenTitular").
		Preload("SenAsignado").
		Where("gestion = ? AND mes = ?", gestion, mes).
		Find(&list).Error
	return list, err
}

func (r *CupoDerechoItemRepository) FindByID(id string) (*models.CupoDerechoItem, error) {
	var v models.CupoDerechoItem
	err := r.db.
		Preload("SenTitular").
		Preload("SenAsignado").
		First(&v, "id = ?", id).
		Error
	return &v, err
}

func (r *CupoDerechoItemRepository) DeleteUnscoped(v *models.CupoDerechoItem) error {
	return r.db.Unscoped().Delete(v).Error
}

func (r *CupoDerechoItemRepository) FindByCupoDerechoID(cupoDerechoID string) ([]models.CupoDerechoItem, error) {
	var list []models.CupoDerechoItem
	err := r.db.
		Preload("SenTitular").
		Preload("SenAsignado").
		Where("cupo_derecho_id = ?", cupoDerechoID).
		Find(&list).
		Error
	return list, err
}
