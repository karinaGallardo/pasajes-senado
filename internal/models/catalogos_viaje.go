package models

type ConceptoViaje struct {
	BaseModel
	Nombre string `gorm:"size:100;not null;unique"`
	Codigo string `gorm:"size:50;not null;unique"`

	TiposSolicitud []TipoSolicitud `gorm:"foreignKey:ConceptoViajeID"`
}

func (ConceptoViaje) TableName() string { return "concepto_viajes" }

type TipoSolicitud struct {
	BaseModel
	ConceptoViajeID string         `gorm:"size:36;not null"`
	ConceptoViaje   *ConceptoViaje `gorm:"foreignKey:ConceptoViajeID"`
	Nombre          string         `gorm:"size:100;not null"`
	Codigo          string         `gorm:"size:50;not null"`

	Ambitos []AmbitoViaje `gorm:"many2many:tipo_solicitud_ambitos;"`
}

func (TipoSolicitud) TableName() string { return "tipo_solicitudes" }

type AmbitoViaje struct {
	BaseModel
	Nombre string `gorm:"size:50;not null;unique"`
	Codigo string `gorm:"size:20;not null;unique"`
}

func (AmbitoViaje) TableName() string { return "ambito_viajes" }

type TipoItinerario struct {
	BaseModel
	Nombre string `gorm:"size:50;not null;unique"`
	Codigo string `gorm:"size:20;not null;unique"`
}

func (TipoItinerario) TableName() string { return "tipo_itinerarios" }
