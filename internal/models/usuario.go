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

	TitularID            *string                    `gorm:"size:36;index"`
	Titular              *Usuario                   `gorm:"foreignKey:TitularID;<-:false"`
	Suplentes            []Usuario                  `gorm:"foreignKey:TitularID"`
	OrigenesAlternativos []UsuarioOrigenAlternativo `gorm:"foreignKey:UsuarioID"`

	FullName string `gorm:"-" json:"full_name"`

	LoginAttempts int  `gorm:"default:0" json:"login_attempts"`
	IsBlocked     bool `gorm:"default:false" json:"is_blocked"`
}

func (u *Usuario) AfterFind(tx *gorm.DB) (err error) {
	u.FullName = u.GetNombreCompleto()
	return
}

func (u Usuario) GetNombreCompleto() string {
	parts := []string{u.Firstname, u.Secondname, u.Lastname, u.Surname}
	var clean []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			clean = append(clean, strings.TrimSpace(p))
		}
	}
	return strings.Join(clean, " ")
}

func (u *Usuario) GetInitials() string {
	var initials string
	if len(u.Firstname) > 0 {
		initials += string([]rune(u.Firstname)[0])
	}
	if len(u.Lastname) > 0 {
		initials += string([]rune(u.Lastname)[0])
	}
	return strings.ToUpper(initials)
}

func (u Usuario) GetNombreResumido() string {
	parts := []string{u.Firstname, u.Secondname, u.Lastname}
	var clean []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			clean = append(clean, strings.TrimSpace(p))
		}
	}
	base := strings.Join(clean, " ")
	if strings.TrimSpace(u.Surname) != "" {
		initial := string([]rune(strings.TrimSpace(u.Surname))[0])
		return base + " " + strings.ToUpper(initial) + "."
	}
	return base
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

func (u Usuario) GetRolName() string {
	if u.Rol != nil {
		return u.Rol.Nombre
	}
	if u.RolCodigo != nil {
		return *u.RolCodigo
	}
	return u.Tipo
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

func (u *Usuario) IsAlternativo(iata string) bool {
	for _, o := range u.OrigenesAlternativos {
		if o.DestinoIATA == iata {
			return true
		}
	}
	return false
}

// --- Gestión de Tipos y Roles ---

func (u *Usuario) IsTitular() bool {
	return u.Tipo == "SENADOR_TITULAR"
}

func (u *Usuario) IsSuplente() bool {
	return u.Tipo == "SENADOR_SUPLENTE"
}

// --- Predicados de Relación ---

func (u *Usuario) IsManagedBy(authUser *Usuario) bool {
	if authUser == nil || u.EncargadoID == nil {
		return false
	}
	return *u.EncargadoID == authUser.ID
}

func (u *Usuario) IsAssistantOf(senador *Usuario) bool {
	if senador == nil || senador.EncargadoID == nil {
		return false
	}
	return *senador.EncargadoID == u.ID
}

func (u *Usuario) IsOwner(id string) bool {
	return u.ID == id
}

// --- Gestión de Estado ---

func (u *Usuario) Unblock() {
	u.IsBlocked = false
	u.LoginAttempts = 0
}

func (u *Usuario) RecordFailedLogin(maxAttempts int) {
	u.LoginAttempts++
	if u.LoginAttempts >= maxAttempts {
		u.IsBlocked = true
	}
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

	if s.CreatedBy != nil && *s.CreatedBy == u.ID {
		return true
	}

	return false
}

func (u *Usuario) CanEditSolicitud(s Solicitud) bool {
	st := s.GetEstado()

	// All users (including Admin) must follow the state rules for editing.
	// Only editable in these specific states.
	isEditableState := (st == "SOLICITADO" || st == "RECHAZADO" || st == "PARCIALMENTE_APROBADO")

	if !isEditableState {
		return false
	}

	// 1. Admins and Responsables can edit any request in an editable state
	if u.IsAdminOrResponsable() {
		return true
	}

	// 2. Ownership / Assistant Check for regular users and assistants
	// Owner of the solicitation
	if u.ID == s.UsuarioID {
		return true
	}

	// Creator of the solicitation
	if s.CreatedBy != nil && *s.CreatedBy == u.ID {
		return true
	}

	// Assigned assistant (Encargado) for the senator
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

type UserPermissions struct {
	CanChangeRol    bool
	CanChangeOrigin bool
	CanChangeStaff  bool
	CanManageRoutes bool
	CanEditContact  bool

	// Atomismo de completitud
	HasOrigin  bool
	HasCargo   bool
	HasOficina bool
	HasRol     bool
	HasEmail   bool
	HasPhone   bool
}

func (u *Usuario) GetPermissionsFor(authUser *Usuario) UserPermissions {
	if authUser == nil {
		return UserPermissions{}
	}

	isAdmin := authUser.IsAdminOrResponsable()
	isSelf := authUser.ID == u.ID
	isEncargado := u.IsManagedBy(authUser)

	// El contacto lo puede editar el propio usuario, su encargado (asistente) o un admin
	canEditContact := isSelf || isEncargado || isAdmin

	// Atomismo de validación
	hasOrigin := u.OrigenIATA != nil && *u.OrigenIATA != ""
	hasCargo := u.CargoID != nil && *u.CargoID != ""
	hasOficina := u.OficinaID != nil && *u.OficinaID != ""
	hasRol := u.RolCodigo != nil && *u.RolCodigo != ""
	hasEmail := u.Email != ""
	hasPhone := u.Phone != ""

	return UserPermissions{
		CanChangeRol:    isAdmin,
		CanChangeOrigin: isAdmin,
		CanChangeStaff:  isAdmin,
		CanManageRoutes: isAdmin,
		CanEditContact:  canEditContact,
		HasOrigin:       hasOrigin,
		HasCargo:        hasCargo,
		HasOficina:      hasOficina,
		HasRol:          hasRol,
		HasEmail:        hasEmail,
		HasPhone:        hasPhone,
	}
}

func (Usuario) TableName() string {
	return "usuarios"
}
