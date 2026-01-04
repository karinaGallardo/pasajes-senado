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

	SolicitudID *string    `gorm:"size:36;index;default:null"`
	Solicitud   *Solicitud `gorm:"foreignKey:SolicitudID"`

	EsTransferido  bool     `gorm:"default:false"`
	BeneficiarioID *string  `gorm:"size:36;index;default:null"`
	Beneficiario   *Usuario `gorm:"foreignKey:BeneficiarioID"`
	FechaTransfer  *time.Time
	MotivoTransfer string `gorm:"size:255"`

	FechaDesde *time.Time
	FechaHasta *time.Time
}

func (AsignacionVoucher) TableName() string {
	return "asignaciones_voucher"
}
