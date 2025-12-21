package models

type Permiso struct {
	Codigo      string `gorm:"primaryKey;size:100;not null"`
	Nombre      string `gorm:"size:100;not null"`
	Descripcion string `gorm:"size:255"`

	Roles []*Rol `gorm:"many2many:rol_permisos;"`
}

func (Permiso) TableName() string {
	return "permisos"
}
