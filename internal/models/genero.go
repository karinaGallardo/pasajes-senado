package models

type Genero struct {
	Codigo      string `gorm:"primaryKey;size:50;not null"`
	Nombre      string `gorm:"uniqueIndex;size:50;not null"`
	Descripcion string `gorm:"size:255"`
}

func (Genero) TableName() string {
	return "generos"
}
