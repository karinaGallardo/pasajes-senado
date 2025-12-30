package models

type TipoSolicitud struct {
	BaseModel
	ConceptoViajeID string         `gorm:"size:36;not null"`
	ConceptoViaje   *ConceptoViaje `gorm:"foreignKey:ConceptoViajeID"`
	Nombre          string         `gorm:"size:100;not null"`
	Codigo          string         `gorm:"size:50;not null"`

	Ambitos []AmbitoViaje `gorm:"many2many:tipo_solicitud_ambitos;"`
}

func (TipoSolicitud) TableName() string { return "tipo_solicitudes" }
