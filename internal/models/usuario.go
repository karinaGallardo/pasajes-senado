package models

import (
	"strings"
)

type Usuario struct {
	BaseModel

	CI       string `gorm:"size:20;index"`
	Username string `gorm:"uniqueIndex;size:100;not null"`
	Email    string `gorm:"size:255"`

	Firstname  string `gorm:"size:100;not null"`
	Secondname string `gorm:"size:100"`
	Lastname   string `gorm:"size:100;not null"`
	Surname    string `gorm:"size:100"`

	Phone   string `gorm:"size:50"`
	Address string `gorm:"size:255"`

	GeneroID *string `gorm:"size:36;index"`
	Genero   *Genero `gorm:"foreignKey:GeneroID"`

	RolID *string `gorm:"size:36;index"`
	Rol   *Rol    `gorm:"foreignKey:RolID"`
}

func (u *Usuario) GetNombreCompleto() string {
	parts := []string{u.Firstname, u.Secondname, u.Lastname, u.Surname}
	var clean []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			clean = append(clean, strings.TrimSpace(p))
		}
	}
	return strings.Join(clean, " ")
}

func (Usuario) TableName() string {
	return "usuarios"
}
