package models

import "time"

type Compensacion struct {
	BaseModel
	Codigo      string `gorm:"size:20;uniqueIndex"`
	Correlativo string `gorm:"size:50"`
	Nombre      string `gorm:"size:255;not null"`

	FechaInicio time.Time `gorm:"not null;type:timestamp"`
	FechaFin    time.Time `gorm:"not null;type:timestamp"`

	MesCompensacion string `gorm:"size:20"`

	FuncionarioID string   `gorm:"size:36;not null;index"`
	Funcionario   *Usuario `gorm:"foreignKey:FuncionarioID"`

	Cargo        string `gorm:"size:100"`
	Departamento string `gorm:"size:100"`
	Informe      string `gorm:"size:100"`

	Estado string `gorm:"size:50;default:'BORRADOR'"`
	Glosa  string `gorm:"type:text"`

	Retencion float64 `gorm:"type:decimal(15,2);default:0"`
	Total     float64 `gorm:"type:decimal(15,2);default:0"`
}

func (Compensacion) TableName() string {
	return "compensaciones"
}
