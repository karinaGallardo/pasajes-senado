package models

import (
	"strings"
)

type Usuario struct {
	BaseModel

	CI       string `gorm:"size:20;uniqueIndex"`
	Username string `gorm:"uniqueIndex;size:100;not null"`
	Email    string `gorm:"size:255"`

	Firstname  string `gorm:"size:100;not null"`
	Secondname string `gorm:"size:100"`
	Lastname   string `gorm:"size:100;not null"`
	Surname    string `gorm:"size:100"`

	Phone   string `gorm:"size:50"`
	Address string `gorm:"size:255"`

	GeneroCodigo *string `gorm:"size:50;index"`
	Genero       *Genero `gorm:"foreignKey:GeneroCodigo"`

	Tipo string `gorm:"size:50;index;default:'FUNCIONARIO'"`

	OrigenCode *string `gorm:"size:4;default:null"`
	Origen     *Ciudad `gorm:"foreignKey:OrigenCode"`

	DepartamentoCode *string       `gorm:"size:5;default:null"`
	Departamento     *Departamento `gorm:"foreignKey:DepartamentoCode"`

	RolCodigo *string `gorm:"size:50;index"`
	Rol       *Rol    `gorm:"foreignKey:RolCodigo"`

	EncargadoID *string  `gorm:"size:36;index"`
	Encargado   *Usuario `gorm:"foreignKey:EncargadoID"`

	CargoID *string `gorm:"size:36;index"`
	Cargo   *Cargo  `gorm:"foreignKey:CargoID"`

	OficinaID *string  `gorm:"size:36;index"`
	Oficina   *Oficina `gorm:"foreignKey:OficinaID"`

	TitularID *string   `gorm:"size:36;index"`
	Titular   *Usuario  `gorm:"foreignKey:TitularID"`
	Suplentes []Usuario `gorm:"foreignKey:TitularID"`
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

func (u *Usuario) GetOrigenCode() string {
	if u.OrigenCode == nil {
		return ""
	}
	return *u.OrigenCode
}

func (u *Usuario) GetSuplente() *Usuario {
	if len(u.Suplentes) > 0 {
		return &u.Suplentes[0]
	}
	return nil
}

func (Usuario) TableName() string {
	return "usuarios"
}
