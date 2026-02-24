package models

import "time"

type Descargo struct {
	BaseModel
	SolicitudID string     `gorm:"not null;size:36;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`
	UsuarioID   string     `gorm:"size:24;not null"`

	Codigo     string `gorm:"size:20;uniqueIndex"`
	NumeroCite string `gorm:"size:50;index"`

	FechaPresentacion time.Time `gorm:"not null;type:timestamp"`
	Observaciones     string    `gorm:"type:text"`

	// Detalles de Itinerario (FV-05) - Relación granular por conexiones
	DetallesItinerario []DetalleItinerarioDescargo `gorm:"foreignKey:DescargoID"`

	Estado string `gorm:"size:50;default:'EN_REVISION'"`

	Documentos []DocumentoDescargo `gorm:"foreignKey:DescargoID"`

	// Detalle opcional para informes oficiales (PV-06)
	Oficial *DescargoOficial `gorm:"foreignKey:DescargoID"`
}

func (Descargo) TableName() string {
	return "descargos"
}
