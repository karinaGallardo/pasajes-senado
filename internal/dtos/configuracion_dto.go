package dtos

type UpdateConfiguracionRequest struct {
	Clave string `form:"clave" binding:"required"`
	Valor string `form:"valor" binding:"required"`
}
