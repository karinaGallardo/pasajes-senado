package models

type Oficina struct {
	BaseModel
	Codigo      int     `gorm:"unique;not null"`
	Detalle     string  `gorm:"size:200;not null"`
	Interno     int     `gorm:"default:0"`
	Area        string  `gorm:"size:100"`
	Presupuesto float64 `gorm:"type:decimal(15,2);default:0"`
}

func (Oficina) TableName() string {
	return "oficinas"
}
