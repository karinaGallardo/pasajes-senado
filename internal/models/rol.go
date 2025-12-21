package models

type Rol struct {
	Codigo      string `gorm:"primaryKey;size:50;not null"`
	Nombre      string `gorm:"size:100;not null"`
	Descripcion string `gorm:"size:255"`

	Permisos []*Permiso `gorm:"many2many:rol_permisos;"`
}

func (Rol) TableName() string {
	return "roles"
}
