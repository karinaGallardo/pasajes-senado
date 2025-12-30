package models

import "time"

type AsignacionVoucher struct {
	BaseModel
	UsuarioID string   `gorm:"size:36;not null;index"`
	Usuario   *Usuario `gorm:"foreignKey:UsuarioID"`

	Gestion int    `gorm:"not null;index"`
	Mes     int    `gorm:"not null;index"`
	Semana  string `gorm:"size:20;index"`

	CupoID string `gorm:"size:36;index"`
	Cupo   *Cupo  `gorm:"foreignKey:CupoID"`

	Estado      string  `gorm:"size:50;default:'DISPONIBLE'"`
	SolicitudID *string `gorm:"size:36;index;default:null"`

	EsTransferido   bool    `gorm:"default:false"`
	UsuarioOrigenID *string `gorm:"size:36;index;default:null"`
	FechaTransfer   *time.Time
	MotivoTransfer  string `gorm:"size:255"`
}

func (AsignacionVoucher) TableName() string {
	return "asignaciones_voucher"
}
