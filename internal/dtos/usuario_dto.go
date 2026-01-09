package dtos

type UpdateUsuarioRequest struct {
	RolCodigo   string `form:"rol_codigo"`
	OrigenIATA  string `form:"origen"`
	EncargadoID string `form:"encargado_id"`
}
