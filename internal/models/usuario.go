package models

import (
	"strings"

	"gorm.io/gorm"
)

type Usuario struct {
	BaseModel

	CI       string `gorm:"size:20;uniqueIndex" json:"ci"`
	Username string `gorm:"uniqueIndex;size:100;not null"`
	Email    string `gorm:"size:255"`

	Firstname  string `gorm:"size:100;not null"`
	Secondname string `gorm:"size:100"`
	Lastname   string `gorm:"size:100;not null"`
	Surname    string `gorm:"size:100"`

	Phone   string `gorm:"size:50"`
	Address string `gorm:"size:255"`

	GeneroCodigo *string `gorm:"size:50;index"`
	Genero       *Genero `gorm:"foreignKey:GeneroCodigo;<-:false"`

	Tipo string `gorm:"size:50;index;default:'FUNCIONARIO'"`

	OrigenIATA *string  `gorm:"size:5;default:null"`
	Origen     *Destino `gorm:"foreignKey:OrigenIATA;references:IATA;<-:false"`

	DepartamentoCode *string       `gorm:"size:5;default:null"`
	Departamento     *Departamento `gorm:"foreignKey:DepartamentoCode;<-:false"`

	RolCodigo *string `gorm:"size:50;index"`
	Rol       *Rol    `gorm:"foreignKey:RolCodigo;<-:false"`

	EncargadoID *string  `gorm:"size:36;index"`
	Encargado   *Usuario `gorm:"foreignKey:EncargadoID;<-:false"`

	CargoID *string `gorm:"size:36;index"`
	Cargo   *Cargo  `gorm:"foreignKey:CargoID;<-:false"`

	OficinaID *string  `gorm:"size:36;index"`
	Oficina   *Oficina `gorm:"foreignKey:OficinaID;<-:false"`

	TitularID *string   `gorm:"size:36;index"`
	Titular   *Usuario  `gorm:"foreignKey:TitularID;<-:false"`
	Suplentes []Usuario `gorm:"foreignKey:TitularID"`

	FullName string `gorm:"-" json:"full_name"`

	LoginAttempts int  `gorm:"default:0" json:"login_attempts"`
	IsBlocked     bool `gorm:"default:false" json:"is_blocked"`
}

func (u *Usuario) AfterFind(tx *gorm.DB) (err error) {
	u.FullName = u.GetNombreCompleto()
	return
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

func (u *Usuario) GetOrigenIATA() string {
	if u.OrigenIATA == nil {
		return ""
	}
	return *u.OrigenIATA
}

func (u *Usuario) GetOrigenNombre() string {
	if u.Origen == nil {
		return ""
	}
	return u.Origen.Ciudad
}

func (u *Usuario) IsAdmin() bool {
	return u.RolCodigo != nil && *u.RolCodigo == "ADMIN"
}

func (u *Usuario) IsResponsable() bool {
	return u.RolCodigo != nil && *u.RolCodigo == "RESPONSABLE"
}

func (u *Usuario) IsSenador() bool {
	return u.RolCodigo != nil && *u.RolCodigo == "SENADOR"
}

func (u *Usuario) IsAdminOrResponsable() bool {
	return u.IsAdmin() || u.IsResponsable()
}

func (u *Usuario) CanManagePasajes(s Solicitud) bool {
	return u.IsAdminOrResponsable()
}

func (u *Usuario) CanMarkUsado(s Solicitud) bool {
	if u.IsAdminOrResponsable() {
		return true
	}

	if u.ID == s.UsuarioID {
		return true
	}

	if s.Usuario.EncargadoID != nil && *s.Usuario.EncargadoID == u.ID {
		return true
	}

	return false
}

func (u *Usuario) CanEditSolicitud(s Solicitud) bool {
	if s.EstadoSolicitudCodigo != nil && *s.EstadoSolicitudCodigo != "SOLICITADO" {
		return false
	}

	if u.IsAdminOrResponsable() {
		return true
	}

	if u.ID == s.UsuarioID {
		return true
	}

	if s.Usuario.EncargadoID != nil && *s.Usuario.EncargadoID == u.ID {
		return true
	}

	return false
}

func (u *Usuario) CanApproveReject() bool {
	return u.IsAdminOrResponsable()
}

func (u *Usuario) CanCreateSolicitudFor(targetUser *Usuario) bool {
	if u.IsAdminOrResponsable() {
		return true
	}

	if u.ID == targetUser.ID {
		return true
	}

	if targetUser.EncargadoID != nil && *targetUser.EncargadoID == u.ID {
		return true
	}

	return false
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
