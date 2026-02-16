package models

type ZonaViatico struct {
	BaseModel
	Nombre string `gorm:"size:100;unique;not null"`
}

func (ZonaViatico) TableName() string {
	return "zonas_viatico"
}
