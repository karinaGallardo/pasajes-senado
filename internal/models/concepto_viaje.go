package models

type ConceptoViaje struct {
	BaseModel
	Nombre string `gorm:"size:100;not null;unique"`
	Codigo string `gorm:"size:50;not null;unique"`

	TiposSolicitud []TipoSolicitud `gorm:"foreignKey:ConceptoViajeID"`
}

func (ConceptoViaje) TableName() string { return "concepto_viajes" }
