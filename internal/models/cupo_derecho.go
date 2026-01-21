package models

import (
	"gorm.io/gorm"
)

type CupoDerecho struct {
	BaseModel
	SenTitularID string            `gorm:"size:36;not null;index;comment:Senador dueño del cupo por derecho"`
	SenTitular   *Usuario          `gorm:"foreignKey:SenTitularID"`
	Gestion      int               `gorm:"not null;index"`
	Mes          int               `gorm:"not null;index"`
	TotalSemanas int               `gorm:"not null"`
	CupoTotal    int               `gorm:"not null"`
	CupoUsado    int               `gorm:"default:0"`
	Items        []CupoDerechoItem `gorm:"foreignKey:CupoDerechoID"`
	Saldo        int               `gorm:"-"`
}

func (c *CupoDerecho) AfterFind(tx *gorm.DB) (err error) {
	c.Saldo = c.CupoTotal - c.CupoUsado
	return
}

func (CupoDerecho) TableName() string {
	return "cupos_derecho"
}
