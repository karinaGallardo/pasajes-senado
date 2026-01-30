package models

type Aerolinea struct {
	BaseModel
	Nombre string `gorm:"size:100;not null;unique"`
	Sigla  string `gorm:"size:20"`
	Estado bool   `gorm:"default:true"`
}

func (Aerolinea) TableName() string { return "aerolineas" }
