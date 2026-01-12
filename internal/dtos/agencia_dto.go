package dtos

type CreateAgenciaRequest struct {
	Nombre   string `form:"nombre" binding:"required"`
	Telefono string `form:"telefono"`
	Estado   string `form:"estado"`
}

type UpdateAgenciaRequest struct {
	Nombre   string `form:"nombre" binding:"required"`
	Telefono string `form:"telefono"`
	Estado   string `form:"estado"`
}
