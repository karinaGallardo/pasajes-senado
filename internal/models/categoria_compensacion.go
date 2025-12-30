package models

type CategoriaCompensacion struct {
	BaseModel
	Departamento string  `gorm:"size:100;not null;uniqueIndex:idx_dep_tipo"`
	TipoSenador  string  `gorm:"size:50;not null;uniqueIndex:idx_dep_tipo"`
	Monto        float64 `gorm:"type:decimal(10,2);not null"`
}

func (CategoriaCompensacion) TableName() string {
	return "categorias_compensacion"
}
