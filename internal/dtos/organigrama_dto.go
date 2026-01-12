package dtos

type CreateCargoRequest struct {
	Codigo      string `form:"codigo" binding:"required"`
	Descripcion string `form:"descripcion" binding:"required"`
	Categoria   string `form:"categoria" binding:"required"`
}

type CreateOficinaRequest struct {
	Codigo      string `form:"codigo" binding:"required"`
	Detalle     string `form:"detalle" binding:"required"`
	Area        string `form:"area"`
	Presupuesto string `form:"presupuesto"`
}
