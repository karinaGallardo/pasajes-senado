package models

import "time"

type Descargo struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	UsuarioID   string     `gorm:"size:24;not null"`

	Codigo     string `gorm:"size:20;uniqueIndex"`
	NumeroCite string `gorm:"size:50;index"`

	FechaPresentacion  time.Time `gorm:"not null"`
	InformeActividades string    `gorm:"type:text"`

	MontoDevolucion float64 `gorm:"type:decimal(10,2);default:0"`
	Observaciones   string  `gorm:"type:text"`

	Estado string `gorm:"size:50;default:'EN_REVISION'"`

	Documentos []DocumentoDescargo `gorm:"foreignKey:DescargoID"`
}

func (Descargo) TableName() string {
	return "descargos"
}
