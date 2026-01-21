package dtos

type UpdateUsuarioRequest struct {
	RolCodigo   string `form:"rol_codigo"`
	OrigenIATA  string `form:"origen"`
	EncargadoID string `form:"encargado_id"`
	Email       string `form:"email"`
	Phone       string `form:"phone"`
}

type UpdateUserOriginRequest struct {
	OrigenCode string `form:"origen_code" binding:"required"`
}
