package models

import (
	"time"
)

type ViaticoPermissions struct {
	CanEdit   bool
	CanDelete bool
	CanPrint  bool
}

type Viatico struct {
	BaseModel
	UsuarioID string  `gorm:"size:36;not null"`
	Usuario   Usuario `gorm:"constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;<-:false"`

	SolicitudID string     `gorm:"size:36;not null;uniqueIndex"`
	Solicitud   *Solicitud `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;<-:false"`

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

	// Contexto de runtime (no persistido)
	authUser    *Usuario            `gorm:"-"`
	Permissions *ViaticoPermissions `gorm:"-"`
}

func (Viatico) TableName() string {
	return "viaticos"
}

func (v Viatico) GetPermissions(u ...*Usuario) ViaticoPermissions {
	user := v.getAuthUser(u...)
	if user == nil {
		return ViaticoPermissions{}
	}

	canMod := user.IsAdminOrResponsable() && v.Estado == "BORRADOR"

	return ViaticoPermissions{
		CanEdit:   canMod,
		CanDelete: canMod,
		CanPrint:  v.Estado != "ANULADO",
	}
}

func (v *Viatico) HydratePermissions(u ...*Usuario) {
	if len(u) > 0 {
		v.authUser = u[0]
	}
	p := v.GetPermissions()
	v.Permissions = &p
}

func (v Viatico) getAuthUser(u ...*Usuario) *Usuario {
	if len(u) > 0 {
		return u[0]
	}
	return v.authUser
}
