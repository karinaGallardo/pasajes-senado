package models

import (
	"gorm.io/gorm"
)

type Cupo struct {
	BaseModel
	SenadorID    string  `gorm:"size:36;index"`
	Senador      Usuario `gorm:"foreignKey:SenadorID"`
	Gestion      int     `gorm:"not null;index"`
	Mes          int     `gorm:"not null;index"`
	TotalSemanas int     `gorm:"not null"`
	CupoTotal    int     `gorm:"not null"`
	CupoUsado    int     `gorm:"default:0"`
	Saldo        int     `gorm:"-"`
}

func (c *Cupo) AfterFind(tx *gorm.DB) (err error) {
	c.Saldo = c.CupoTotal - c.CupoUsado
	return
}

func (Cupo) TableName() string {
	return "cupos"
}
