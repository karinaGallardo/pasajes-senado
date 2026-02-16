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

type CreateCategoriaViaticoRequest struct {
	Nombre        string  `form:"nombre" binding:"required"`
	Codigo        int     `form:"codigo" binding:"required"`
	Monto         float64 `form:"monto" binding:"required"`
	Moneda        string  `form:"moneda" binding:"required"`
	ZonaViaticoID string  `form:"zona_viatico_id" binding:"required"`
}

type CreateZonaViaticoRequest struct {
	Nombre string `form:"nombre" binding:"required"`
}
