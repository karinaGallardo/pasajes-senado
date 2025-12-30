package models

type Cargo struct {
	BaseModel
	Codigo      int    `gorm:"unique;not null"`
	Descripcion string `gorm:"size:200;not null"`
	Categoria   int    `gorm:"default:0"`
}

func (Cargo) TableName() string {
	return "cargos"
}
