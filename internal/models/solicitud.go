package models

import "time"

type Solicitud struct {
	BaseModel
	Codigo    string  `gorm:"size:8;uniqueIndex"`
	UsuarioID string  `gorm:"size:24;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	VoucherID         *string            `gorm:"size:36;index;default:null"`
	AsignacionVoucher *AsignacionVoucher `gorm:"foreignKey:VoucherID"`

	TipoSolicitudID string         `gorm:"size:36;not null;index"`
	TipoSolicitud   *TipoSolicitud `gorm:"foreignKey:TipoSolicitudID"`

	AmbitoViajeID string       `gorm:"size:36;not null;index"`
	AmbitoViaje   *AmbitoViaje `gorm:"foreignKey:AmbitoViajeID"`

	TipoItinerarioID string          `gorm:"size:36;not null;index"`
	TipoItinerario   *TipoItinerario `gorm:"foreignKey:TipoItinerarioID"`

	OrigenCode string `gorm:"size:4;not null"`
	Origen     Ciudad `gorm:"foreignKey:OrigenCode"`

	DestinoCode string `gorm:"size:4;not null"`
	Destino     Ciudad `gorm:"foreignKey:DestinoCode"`

	FechaIda    *time.Time `gorm:"default:null;type:timestamp"`
	FechaVuelta *time.Time `gorm:"default:null;type:timestamp"`

	Motivo string `gorm:"type:text"`

	AerolineaSugerida string `gorm:"size:100"`

	EstadoSolicitudCodigo *string          `gorm:"size:20;index;default:'SOLICITADO'"`
	EstadoSolicitud       *EstadoSolicitud `gorm:"foreignKey:EstadoSolicitudCodigo;references:Codigo"`

	Pasajes []Pasaje `gorm:"foreignKey:SolicitudID"`

	Viaticos []Viatico `gorm:"foreignKey:SolicitudID"`

	Descargo *Descargo `gorm:"foreignKey:SolicitudID"`
}

func (Solicitud) TableName() string {
	return "solicitudes"
}

func (s Solicitud) GetEstado() string {
	if s.EstadoSolicitudCodigo == nil {
		return "SOLICITADO"
	}
	return *s.EstadoSolicitudCodigo
}

func (s Solicitud) GetEstadoCodigo() string {
	if s.EstadoSolicitudCodigo == nil {
		return ""
	}
	return *s.EstadoSolicitudCodigo
}
