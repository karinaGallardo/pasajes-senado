package models

import "time"

type CupoDerechoItem struct {
	BaseModel
	SenTitularID string   `gorm:"size:36;not null;index;comment:Senador dueño del cupo por derecho"`
	SenTitular   *Usuario `gorm:"foreignKey:SenTitularID"`

	SenAsignadoID string   `gorm:"size:36;not null;index;comment:Senador que tiene el derecho de uso actual"`
	SenAsignado   *Usuario `gorm:"foreignKey:SenAsignadoID"`

	EsTransferido  bool       `gorm:"default:false;comment:Indica si el derecho ha sido transferido"`
	FechaTransfer  *time.Time `gorm:"type:timestamp;comment:Fecha de la transferencia"`
	MotivoTransfer string     `gorm:"size:255;comment:Motivo de la transferencia"`

	Gestion int    `gorm:"not null;index"`
	Mes     int    `gorm:"not null;index"`
	Semana  string `gorm:"size:50;index"`

	CupoDerechoID string       `gorm:"size:36;index;comment:ID del cupo por derecho"`
	CupoDerecho   *CupoDerecho `gorm:"foreignKey:CupoDerechoID"`

	EstadoCupoDerechoCodigo string             `gorm:"size:50;default:'DISPONIBLE';index" json:"Estado"`
	EstadoCupoDerecho       *EstadoCupoDerecho `gorm:"foreignKey:EstadoCupoDerechoCodigo"`

	Solicitudes []Solicitud `gorm:"foreignKey:CupoDerechoItemID"`

	FechaDesde *time.Time `gorm:"type:timestamp;comment:Fecha desde la cual el cupo es válido"`
	FechaHasta *time.Time `gorm:"type:timestamp;comment:Fecha hasta la cual el cupo es válido"`
}

func (CupoDerechoItem) TableName() string {
	return "cupo_derecho_items"
}

func (v CupoDerechoItem) GetSolicitudIda() *Solicitud {
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.TipoItinerario != nil && (s.TipoItinerario.Codigo == "SOLO_IDA" || s.TipoItinerario.Codigo == "IDA_VUELTA") {
			if s.FechaIda != nil && (s.EstadoSolicitudCodigo == nil || *s.EstadoSolicitudCodigo != "RECHAZADO") {
				return s
			}
		}
	}
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.FechaIda != nil && (s.EstadoSolicitudCodigo == nil || *s.EstadoSolicitudCodigo != "RECHAZADO") {
			return s
		}
	}
	return nil
}

func (v CupoDerechoItem) GetSolicitudVuelta() *Solicitud {
	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.TipoItinerario != nil && (s.TipoItinerario.Codigo == "SOLO_VUELTA") {
			if s.FechaIda != nil && (s.EstadoSolicitudCodigo == nil || *s.EstadoSolicitudCodigo != "RECHAZADO") {
				return s
			}
		}
	}

	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.TipoItinerario != nil && (s.TipoItinerario.Codigo == "IDA_VUELTA") {
			if s.FechaVuelta != nil && (s.EstadoSolicitudCodigo == nil || *s.EstadoSolicitudCodigo != "RECHAZADO") {
				return s
			}
		}
	}

	for i := range v.Solicitudes {
		s := &v.Solicitudes[i]
		if s.FechaVuelta != nil && (s.EstadoSolicitudCodigo == nil || *s.EstadoSolicitudCodigo != "RECHAZADO") {
			return s
		}
	}
	return nil
}

func (v CupoDerechoItem) GetDescargo() *Descargo {
	for i := range v.Solicitudes {
		if v.Solicitudes[i].Descargo != nil {
			return v.Solicitudes[i].Descargo
		}
	}
	return nil
}

func (v CupoDerechoItem) IsVencido() bool {
	if v.FechaHasta == nil {
		return false
	}
	expirationTime := v.FechaHasta.Add(24 * time.Hour)
	return time.Now().After(expirationTime)
}

func (v CupoDerechoItem) IsActiveWeek() bool {
	if v.FechaDesde == nil || v.FechaHasta == nil {
		return false
	}
	now := time.Now()
	return now.After(*v.FechaDesde) && now.Before(v.FechaHasta.Add(24*time.Hour))
}

func (v CupoDerechoItem) IsDisponible() bool {
	return v.EstadoCupoDerechoCodigo == "DISPONIBLE"
}
