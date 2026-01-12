package dtos

type GenerarCupoRequest struct {
	Gestion string `form:"gestion"`
	Mes     string `form:"mes"`
}

type ResetCupoRequest struct {
	Gestion string `form:"gestion"`
	Mes     string `form:"mes"`
}

type TransferirVoucherRequest struct {
	VoucherID string `form:"voucher_id" binding:"required"`
	DestinoID string `form:"destino_id" binding:"required"`
	Motivo    string `form:"motivo"`
	Gestion   string `form:"gestion"`
	Mes       string `form:"mes"`
}
