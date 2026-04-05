package dtos

import (
	"sistema-pasajes/internal/models"
)

// SyncResult representa el resultado de una operación de sincronización con RRHH
type SyncResult struct {
	Count     int
	Conflicts []string
}

// UserEditContext contiene toda la información necesaria para renderizar la vista de edición de un usuario
type UserEditContext struct {
	Usuario      *models.Usuario
	Roles        []models.Rol
	Destinos     []models.Destino
	Funcionarios []models.Usuario
	Cargos       []models.Cargo
	Oficinas     []models.Oficina
	Permissions  map[string]bool
}

// UpdateUsuarioRequest representa los datos recibidos al actualizar un perfil de usuario
type UpdateUsuarioRequest struct {
	RolCodigo            string   `form:"rol_codigo"`
	OrigenIATA           string   `form:"origen"`
	EncargadoID          string   `form:"encargado_id"`
	Email                string   `form:"email"`
	Phone                string   `form:"phone"`
	OrigenesAlternativos []string `form:"origenes_alternativos"`
}

// UpdateUserOriginRequest representa los datos para actualizar solo el origen de un usuario
type UpdateUserOriginRequest struct {
	OrigenCode string `form:"origen_code" json:"origen_code"`
}
