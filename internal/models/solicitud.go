package models

import "time"

type Solicitud struct {
	BaseModel
	Codigo    string  `gorm:"size:12;uniqueIndex"`
	UsuarioID string  `gorm:"size:24;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	CupoDerechoItemID *string          `gorm:"size:36;index;default:null"`
	CupoDerechoItem   *CupoDerechoItem `gorm:"foreignKey:CupoDerechoItemID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;<-:false"`

	TipoSolicitudID string         `gorm:"size:36;not null;index"`
	TipoSolicitud   *TipoSolicitud `gorm:"foreignKey:TipoSolicitudID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	AmbitoViajeID string       `gorm:"size:36;not null;index"`
	AmbitoViaje   *AmbitoViaje `gorm:"foreignKey:AmbitoViajeID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	TipoItinerarioID string          `gorm:"size:36;not null;index"`
	TipoItinerario   *TipoItinerario `gorm:"foreignKey:TipoItinerarioID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	OrigenIATA string   `gorm:"size:5;not null"`
	Origen     *Destino `gorm:"foreignKey:OrigenIATA;references:IATA;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	DestinoIATA string   `gorm:"size:5;not null"`
	Destino     *Destino `gorm:"foreignKey:DestinoIATA;references:IATA;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	FechaIda    *time.Time `gorm:"default:null;type:timestamp"`
	FechaVuelta *time.Time `gorm:"default:null;type:timestamp"`

	Motivo string `gorm:"type:text"`

	AerolineaSugerida string `gorm:"size:100"`

	EstadoSolicitudCodigo *string          `gorm:"size:20;index;default:'SOLICITADO'"`
	EstadoSolicitud       *EstadoSolicitud `gorm:"foreignKey:EstadoSolicitudCodigo;references:Codigo;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	Pasajes []Pasaje `gorm:"foreignKey:SolicitudID"`

	Viaticos []Viatico `gorm:"foreignKey:SolicitudID"`

	Descargo *Descargo `gorm:"foreignKey:SolicitudID"`

	Autorizacion string `gorm:"size:100;index"`

	// Flags for Printing logic
	VueltaSeparada bool `gorm:"default:false"`
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
