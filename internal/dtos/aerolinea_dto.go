package dtos

type CreateAerolineaRequest struct {
	Nombre string `form:"nombre" binding:"required"`
	Estado string `form:"estado"` // checkbox sends "on" or nothing
}

type UpdateAerolineaRequest struct {
	Nombre string `form:"nombre" binding:"required"`
	Estado string `form:"estado"`
}
