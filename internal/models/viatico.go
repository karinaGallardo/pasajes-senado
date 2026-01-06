package models

import "time"

type Viatico struct {
	BaseModel
	UsuarioID string  `gorm:"size:36;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;"`

	SolicitudID string     `gorm:"size:36;not null;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`

	Codigo string `gorm:"size:20;uniqueIndex"`

	FechaAsignacion time.Time `gorm:"not null;type:timestamp"`

	Lugar string `gorm:"size:200"`
	Glosa string `gorm:"type:text"`

	TipoTransporte string `gorm:"size:100"`

	MontoTotal   float64 `gorm:"type:decimal(10,2);not null"`
	MontoRC_IVA  float64 `gorm:"type:decimal(10,2);not null"`
	MontoLiquido float64 `gorm:"type:decimal(10,2);not null"`

	TieneGastosRep       bool
	MontoGastosRep       float64 `gorm:"type:decimal(10,2);default:0"`
	MontoRetencionGastos float64 `gorm:"type:decimal(10,2);default:0"`
	MontoLiquidoGastos   float64 `gorm:"type:decimal(10,2);default:0"`

	Estado string `gorm:"size:50;default:'BORRADOR';index"`

	Detalles []DetalleViatico `gorm:"foreignKey:ViaticoID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (Viatico) TableName() string {
	return "viaticos"
}
