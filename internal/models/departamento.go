package models

type Departamento struct {
	Codigo string `gorm:"primaryKey;size:5;not null"`
	Nombre string `gorm:"column:nombre;size:100;not null;unique"`
}

func (Departamento) TableName() string {
	return "departamentos"
}
