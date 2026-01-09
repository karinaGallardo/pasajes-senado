package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type AsignacionVoucherRepository struct {
	db *gorm.DB
}

func NewAsignacionVoucherRepository() *AsignacionVoucherRepository {
	return &AsignacionVoucherRepository{db: configs.DB}
}

func (r *AsignacionVoucherRepository) WithTx(tx *gorm.DB) *AsignacionVoucherRepository {
	return &AsignacionVoucherRepository{db: tx}
}

func (r *AsignacionVoucherRepository) WithContext(ctx context.Context) *AsignacionVoucherRepository {
	return &AsignacionVoucherRepository{db: r.db.WithContext(ctx)}
}

func (r *AsignacionVoucherRepository) GetDB() *gorm.DB {
	return r.db
}

func (r *AsignacionVoucherRepository) Create(voucher *models.AsignacionVoucher) error {
	return r.db.Create(voucher).Error
}

func (r *AsignacionVoucherRepository) FindByTitularAndPeriodo(titularID string, gestion, mes int) ([]models.AsignacionVoucher, error) {
	var list []models.AsignacionVoucher
	err := r.db.Preload("Solicitudes.Pasajes").Preload("Solicitudes.Descargo").Preload("Solicitudes.TipoItinerario").Where("senador_id = ? AND gestion = ? AND mes = ?", titularID, gestion, mes).Find(&list).Error
	return list, err
}

func (r *AsignacionVoucherRepository) FindByHolderAndPeriodo(userID string, gestion, mes int) ([]models.AsignacionVoucher, error) {
	var list []models.AsignacionVoucher
	err := r.db.Preload("Solicitudes.Pasajes").Preload("Solicitudes.Descargo").Preload("Solicitudes.TipoItinerario").Where("(beneficiario_id = ? OR (beneficiario_id IS NULL AND senador_id = ?)) AND gestion = ? AND mes = ?", userID, userID, gestion, mes).Find(&list).Error
	return list, err
}

func (r *AsignacionVoucherRepository) FindByHolderAndGestion(userID string, gestion int) ([]models.AsignacionVoucher, error) {
	var list []models.AsignacionVoucher
	err := r.db.Preload("Solicitudes.Pasajes").Preload("Solicitudes.Descargo").Preload("Solicitudes.TipoItinerario").Where("(beneficiario_id = ? OR (beneficiario_id IS NULL AND senador_id = ?)) AND gestion = ?", userID, userID, gestion).Order("mes asc, semana asc").Find(&list).Error
	return list, err
}

func (r *AsignacionVoucherRepository) FindAvailableByHolderAndPeriodo(userID string, gestion, mes int) (*models.AsignacionVoucher, error) {
	var voucher models.AsignacionVoucher
	err := r.db.Where("(beneficiario_id = ? OR (beneficiario_id IS NULL AND senador_id = ?)) AND gestion = ? AND mes = ? AND estado_voucher_codigo = 'DISPONIBLE'", userID, userID, gestion, mes).First(&voucher).Error
	return &voucher, err
}

func (r *AsignacionVoucherRepository) Update(voucher *models.AsignacionVoucher) error {
	return r.db.Save(voucher).Error
}

func (r *AsignacionVoucherRepository) FindByPeriodo(gestion, mes int) ([]models.AsignacionVoucher, error) {
	var list []models.AsignacionVoucher
	err := r.db.Preload("Senador").Preload("Beneficiario").Where("gestion = ? AND mes = ?", gestion, mes).Find(&list).Error
	return list, err
}

func (r *AsignacionVoucherRepository) FindByID(id string) (*models.AsignacionVoucher, error) {
	var v models.AsignacionVoucher
	err := r.db.First(&v, "id = ?", id).Error
	return &v, err
}

func (r *AsignacionVoucherRepository) DeleteUnscoped(v *models.AsignacionVoucher) error {
	return r.db.Unscoped().Delete(v).Error
}

func (r *AsignacionVoucherRepository) FindByCupoID(cupoID string) ([]models.AsignacionVoucher, error) {
	var list []models.AsignacionVoucher
	err := r.db.Preload("Senador").Preload("Beneficiario").Where("cupo_id = ?", cupoID).Find(&list).Error
	return list, err
}
