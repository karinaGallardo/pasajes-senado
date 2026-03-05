package repositories

import (
	"context"
	"sistema-pasajes/internal/models"


	"gorm.io/gorm"
)

type TipoSolicitudRepository struct {
	db *gorm.DB
}

func NewTipoSolicitudRepository(db *gorm.DB) *TipoSolicitudRepository {
	return &TipoSolicitudRepository{db: db}
}

func (r *TipoSolicitudRepository) WithContext(ctx context.Context) *TipoSolicitudRepository {
	return &TipoSolicitudRepository{db: r.db.WithContext(ctx)}
}

func (r *TipoSolicitudRepository) FindByID(ctx context.Context, codigo string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := r.db.WithContext(ctx).Preload("ConceptoViaje").First(&tipo, "codigo = ?", codigo).Error
	return &tipo, err
}

func (r *TipoSolicitudRepository) FindByConceptoCodigo(ctx context.Context, conceptoCodigo string) ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.WithContext(ctx).Where("concepto_viaje_codigo = ?", conceptoCodigo).Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAll(ctx context.Context) ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.WithContext(ctx).Preload("ConceptoViaje").Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAmbitosByTipoCodigo(ctx context.Context, tipoCodigo string) ([]models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := r.db.WithContext(ctx).Preload("Ambitos").First(&tipo, "codigo = ?", tipoCodigo).Error
	if err != nil {
		return nil, err
	}
	return tipo.Ambitos, nil
}
func (r *TipoSolicitudRepository) FindByCodigo(ctx context.Context, codigo string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := r.db.WithContext(ctx).Preload("ConceptoViaje").Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *TipoSolicitudRepository) FindByCodigoAndAmbito(ctx context.Context, tipoCodigo, ambitoCodigo string) (*models.TipoSolicitud, *models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := r.db.WithContext(ctx).Preload("ConceptoViaje").
		Preload("Ambitos", "codigo = ?", ambitoCodigo).
		Joins("JOIN tipo_solic_ambitos tsa ON tsa.tipo_solicitud_codigo = tipo_solicitudes.codigo").
		Joins("JOIN ambito_viajes av ON av.codigo = tsa.ambito_viaje_codigo").
		Where("tipo_solicitudes.codigo = ? AND av.codigo = ?", tipoCodigo, ambitoCodigo).
		First(&tipo).Error

	if err != nil {
		return nil, nil, err
	}

	var ambito *models.AmbitoViaje
	if len(tipo.Ambitos) > 0 {
		ambito = &tipo.Ambitos[0]
	}

	return &tipo, ambito, nil
}
