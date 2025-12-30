package models

type AmbitoViaje struct {
	BaseModel
	Nombre string `gorm:"size:50;not null;unique"`
	Codigo string `gorm:"size:20;not null;unique"`
}

func (AmbitoViaje) TableName() string { return "ambito_viajes" }
