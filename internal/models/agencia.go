package models

type Agencia struct {
	BaseModel
	Nombre   string `gorm:"size:200;not null;unique"`
	Estado   bool   `gorm:"default:true"`
	Telefono string `gorm:"size:50"`
}

func (Agencia) TableName() string { return "agencias" }
