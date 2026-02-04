package models

import "time"

type TipoSolicitud struct {
	Codigo              string         `gorm:"primaryKey;size:50;not null" json:"codigo"`
	ConceptoViajeCodigo string         `gorm:"size:50;not null" json:"concepto_viaje_codigo"`
	ConceptoViaje       *ConceptoViaje `gorm:"foreignKey:ConceptoViajeCodigo;references:Codigo" json:"concepto_viaje,omitempty"`
	Nombre              string         `gorm:"size:100;not null" json:"nombre"`

	Ambitos []AmbitoViaje `gorm:"many2many:tipo_solic_ambitos;foreignKey:Codigo;joinForeignKey:TipoSolicitudCodigo;References:Codigo;joinReferences:AmbitoViajeCodigo" json:"ambitos,omitempty"`

	CreatedAt time.Time `gorm:"type:timestamp" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp" json:"updated_at"`
}

func (TipoSolicitud) TableName() string { return "tipo_solicitudes" }
