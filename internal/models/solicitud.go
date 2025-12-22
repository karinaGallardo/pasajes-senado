package models

import "time"

type Solicitud struct {
	BaseModel
	UsuarioID string  `gorm:"size:24;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	TipoSolicitudID string         `gorm:"size:36;not null;index"`
	TipoSolicitud   *TipoSolicitud `gorm:"foreignKey:TipoSolicitudID"`

	AmbitoViajeID string       `gorm:"size:36;not null;index"`
	AmbitoViaje   *AmbitoViaje `gorm:"foreignKey:AmbitoViajeID"`

	TipoItinerarioID string          `gorm:"size:36;not null;index"`
	TipoItinerario   *TipoItinerario `gorm:"foreignKey:TipoItinerarioID"`

	OrigenCode string `gorm:"size:4;not null"`
	Origen     Ciudad `gorm:"foreignKey:OrigenCode"`

	DestinoCode  string    `gorm:"size:4;not null"`
	Destino      Ciudad    `gorm:"foreignKey:DestinoCode"`
	FechaSalida  time.Time `gorm:"not null"`
	FechaRetorno time.Time `gorm:"not null"`
	Motivo       string    `gorm:"type:text"`

	Estado string `gorm:"size:50;default:'SOLICITADO';index"`

	Pasajes []Pasaje `gorm:"foreignKey:SolicitudID"`
}

func (Solicitud) TableName() string {
	return "solicitudes"
}
