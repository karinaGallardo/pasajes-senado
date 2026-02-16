package models

type CategoriaViatico struct {
	BaseModel
	Nombre        string       `gorm:"size:100;not null"`
	Codigo        int          `gorm:"not null"`
	Monto         float64      `gorm:"type:decimal(10,2);not null"`
	Moneda        string       `gorm:"size:10;default:'Bs'"`
	Ubicacion     string       `gorm:"size:50;default:'INTERIOR'"`
	ZonaViaticoID *string      `gorm:"size:36;index"`
	ZonaViatico   *ZonaViatico `gorm:"foreignKey:ZonaViaticoID"`
}

func (CategoriaViatico) TableName() string {
	return "categorias_viatico"
}
