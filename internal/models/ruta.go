package models

type Ruta struct {
	BaseModel
	Tramo     string `gorm:"size:255;not null;unique"`
	Sigla     string `gorm:"size:50"`
	NacInter  string `gorm:"size:50;not null"`
	Ubicacion string `gorm:"size:100"`
	Medio     string `gorm:"size:50;default:'AEREO'"`
}

func (Ruta) TableName() string { return "rutas" }
