package dtos

type CreateRutaRequest struct {
	OrigenIATA  string   `form:"origen_iata" binding:"required"`
	EscalasIATA []string `form:"escalas_iata[]"`
	DestinoIATA string   `form:"destino_iata" binding:"required"`
}

type AddContractRequest struct {
	RutaID      string `form:"ruta_id" binding:"required"`
	AerolineaID string `form:"aerolinea_id" binding:"required"`
	Monto       string `form:"monto" binding:"required,numeric"`
}
