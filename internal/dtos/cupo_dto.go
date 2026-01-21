package dtos

type GenerarCupoRequest struct {
	Gestion string `form:"gestion"`
	Mes     string `form:"mes"`
}

type ResetCupoRequest struct {
	Gestion string `form:"gestion"`
	Mes     string `form:"mes"`
}

type TransferirCupoDerechoItemRequest struct {
	ItemID    string `form:"item_id" binding:"required"`
	DestinoID string `form:"destino_id" binding:"required"`
	Motivo    string `form:"motivo"`
	Gestion   string `form:"gestion"`
	Mes       string `form:"mes"`
	ReturnURL string `form:"return_url"`
}
