package models

type EstadoPasaje struct {
	Codigo      string `gorm:"primaryKey;size:50"`
	Nombre      string `gorm:"size:50;not null"`
	Descripcion string `gorm:"size:255"`
	Color       string `gorm:"size:20"`
	Icon        string `gorm:"size:50;default:'ph ph-ticket'"`
}

func (EstadoPasaje) TableName() string {
	return "estados_pasaje"
}
