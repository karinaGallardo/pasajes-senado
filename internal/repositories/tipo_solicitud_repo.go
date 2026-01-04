package repositories

import (
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

func (r *TipoSolicitudRepository) FindByID(id string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").First(&tipo, "id = ?", id).Error
	return &tipo, err
}

func (r *TipoSolicitudRepository) FindByConceptoID(conceptoID string) ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.Where("concepto_viaje_id = ?", conceptoID).Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAll() ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := r.db.Preload("ConceptoViaje").Find(&tipos).Error
	return tipos, err
}

func (r *TipoSolicitudRepository) FindAmbitosByTipoID(tipoID string) ([]models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := r.db.Preload("Ambitos").First(&tipo, "id = ?", tipoID).Error
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
