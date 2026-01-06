package models

import "time"

type AsignacionVoucher struct {
	BaseModel
	SenadorID string   `gorm:"size:36;not null;index"`
	Senador   *Usuario `gorm:"foreignKey:SenadorID"`

	Gestion int    `gorm:"not null;index"`
	Mes     int    `gorm:"not null;index"`
	Semana  string `gorm:"size:20;index"`

	CupoID string `gorm:"size:36;index"`
	Cupo   *Cupo  `gorm:"foreignKey:CupoID"`

	EstadoVoucherCodigo string         `gorm:"size:50;default:'DISPONIBLE';index" json:"Estado"`
	EstadoVoucher       *EstadoVoucher `gorm:"foreignKey:EstadoVoucherCodigo"`

	Solicitudes []Solicitud `gorm:"foreignKey:VoucherID"`

	EsTransferido  bool       `gorm:"default:false"`
	BeneficiarioID *string    `gorm:"size:36;index;default:null"`
	Beneficiario   *Usuario   `gorm:"foreignKey:BeneficiarioID"`
	FechaTransfer  *time.Time `gorm:"type:timestamp"`
	MotivoTransfer string     `gorm:"size:255"`

	FechaDesde *time.Time `gorm:"type:timestamp"`
	FechaHasta *time.Time `gorm:"type:timestamp"`
}

func (AsignacionVoucher) TableName() string {
	return "asignaciones_voucher"
}

func (v AsignacionVoucher) GetSolicitudIda() *Solicitud {
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

func (v AsignacionVoucher) GetSolicitudVuelta() *Solicitud {
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

func (v AsignacionVoucher) GetDescargo() *Descargo {
	for i := range v.Solicitudes {
		if v.Solicitudes[i].Descargo != nil {
			return v.Solicitudes[i].Descargo
		}
	}
	return nil
}
