package repositories

import (
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/models"
)

type CatalogoRepository struct{}

func NewCatalogoRepository() *CatalogoRepository {
	return &CatalogoRepository{}
}

func (r *CatalogoRepository) FindConceptos() ([]models.ConceptoViaje, error) {
	var conceptos []models.ConceptoViaje
	err := configs.DB.Preload("TiposSolicitud").Preload("TiposSolicitud.Ambitos").Find(&conceptos).Error
	return conceptos, err
}

func (r *CatalogoRepository) FindTipoSolicitudByID(id string) (*models.TipoSolicitud, error) {
	var tipo models.TipoSolicitud
	err := configs.DB.Preload("ConceptoViaje").First(&tipo, "id = ?", id).Error
	return &tipo, err
}

func (r *CatalogoRepository) FindTiposByConceptoID(conceptoID string) ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := configs.DB.Where("concepto_viaje_id = ?", conceptoID).Find(&tipos).Error
	return tipos, err
}

func (r *CatalogoRepository) FindAmbitosByTipoID(tipoID string) ([]models.AmbitoViaje, error) {
	var tipo models.TipoSolicitud
	err := configs.DB.Preload("Ambitos").First(&tipo, "id = ?", tipoID).Error
	if err != nil {
		return nil, err
	}
	return tipo.Ambitos, nil
}

func (r *CatalogoRepository) FindTipoItinerarioByCodigo(codigo string) (*models.TipoItinerario, error) {
	var tipo models.TipoItinerario
	err := configs.DB.Where("codigo = ?", codigo).First(&tipo).Error
	return &tipo, err
}

func (r *CatalogoRepository) FindAllTiposItinerario() ([]models.TipoItinerario, error) {
	var tipos []models.TipoItinerario
	err := configs.DB.Find(&tipos).Error
	return tipos, err
}

func (r *CatalogoRepository) FindAllTiposSolicitud() ([]models.TipoSolicitud, error) {
	var tipos []models.TipoSolicitud
	err := configs.DB.Preload("ConceptoViaje").Find(&tipos).Error
	return tipos, err
}

func (r *CatalogoRepository) FindAllAmbitosViaje() ([]models.AmbitoViaje, error) {
	var ambitos []models.AmbitoViaje
	err := configs.DB.Find(&ambitos).Error
	return ambitos, err
}
