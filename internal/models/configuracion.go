package models

type Configuracion struct {
	BaseModel
	Clave string `gorm:"size:50;unique;not null"`
	Valor string `gorm:"size:255;not null"`
	Tipo  string `gorm:"size:20;default:'STRING'"`
}

func (Configuracion) TableName() string {
	return "configuraciones"
}
