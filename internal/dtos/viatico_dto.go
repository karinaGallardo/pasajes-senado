package dtos

type CreateViaticoRequest struct {
	FechaDesde string `form:"fecha_desde" binding:"required"`
	FechaHasta string `form:"fecha_hasta" binding:"required"`
	Lugar      string `form:"lugar" binding:"required"`
	Dias       string `form:"dias" binding:"required"`
	MontoDia   string `form:"monto_dia" binding:"required"`
	Porcentaje string `form:"porcentaje" binding:"required"`
	GastosRep  string `form:"gastos_rep"`
}
