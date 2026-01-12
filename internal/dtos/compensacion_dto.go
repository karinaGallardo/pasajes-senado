package dtos

type CreateCompensacionRequest struct {
	NombreTramite string `form:"nombre_tramite" binding:"required"`
	FuncionarioID string `form:"funcionario_id" binding:"required"`
	FechaInicio   string `form:"fecha_inicio" binding:"required"`
	FechaFin      string `form:"fecha_fin" binding:"required"`
	Mes           string `form:"mes" binding:"required"`
	Glosa         string `form:"glosa"`
	Total         string `form:"total" binding:"required"`
	Retencion     string `form:"retencion"`
	Informe       string `form:"informe"`
}

type CreateCategoriaCompensacionRequest struct {
	Departamento string `form:"departamento" binding:"required"`
	TipoSenador  string `form:"tipo_senador" binding:"required"`
	Monto        string `form:"monto" binding:"required"`
}
