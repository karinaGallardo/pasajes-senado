package models

type RutaContrato struct {
	BaseModel
	RutaID string `gorm:"size:36;not null;index"`
	Ruta   *Ruta  `gorm:"foreignKey:RutaID"`

	AerolineaID string     `gorm:"size:36;not null;index"`
	Aerolinea   *Aerolinea `gorm:"foreignKey:AerolineaID"`

	MontoReferencial float64 `gorm:"type:decimal(10,2);not null"`
}

func (RutaContrato) TableName() string {
	return "rutas_contrato"
}
