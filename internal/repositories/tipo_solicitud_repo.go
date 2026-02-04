package repositories

import (
	"context"
	"sistema-pasajes/internal/models"

	"sistema-pasajes/internal/configs"

	"gorm.io/gorm"
)

type TipoSolicitudRepository struct {
	db *gorm.DB
}

func NewTipoSolicitudRepository() *TipoSolicitudRepository {
	return &TipoSolicitudRepository{db: configs.DB}
}

func (r *TipoSolicitudRepository) WithContext(ctx context.Context) *TipoSolicitudRepository {
	return &TipoSolicitudRepository{db: r.db.WithContext(ctx)}
}

func (r *TipoSolicitudRepository) FindByID(codigo string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").First(&tipo, "codigo = ?", codigo).Error
	return &tipo, err
}

func (r *TipoSolicitudRepository) FindByConceptoCodigo(conceptoCodigo string) ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.Where("concepto_viaje_codigo = ?", conceptoCodigo).Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAll() ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAmbitosByTipoCodigo(tipoCodigo string) ([]models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("Ambitos").First(&tipo, "codigo = ?", tipoCodigo).Error
	if err != nil {
		return nil, err
	}
	return tipo.Ambitos, nil
}
func (r *TipoSolicitudRepository) FindByCodigo(codigo string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *TipoSolicitudRepository) FindByCodigoAndAmbito(tipoCodigo, ambitoCodigo string) (*models.TipoSolicitud, *models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").
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
