package dtos

type CreateRutaRequest struct {
	Tramo  string `form:"tramo" binding:"required"`
	Sigla  string `form:"sigla" binding:"required"`
	Ambito string `form:"ambito" binding:"required"`
}

type AddContractRequest struct {
	RutaID      string `form:"ruta_id" binding:"required"`
	AerolineaID string `form:"aerolinea_id" binding:"required"`
	Monto       string `form:"monto" binding:"required,numeric"`
}
