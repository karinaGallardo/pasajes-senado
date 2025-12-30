package models

type TipoItinerario struct {
	BaseModel
	Nombre string `gorm:"size:50;not null;unique"`
	Codigo string `gorm:"size:20;not null;unique"`
}

func (TipoItinerario) TableName() string { return "tipo_itinerarios" }
