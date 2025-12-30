package models

type Aerolinea struct {
	BaseModel
	Nombre string `gorm:"size:100;not null;unique"`
	Estado bool   `gorm:"default:true"`
}

func (Aerolinea) TableName() string { return "aerolineas" }
