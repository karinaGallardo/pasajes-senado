package dtos

type ReprogramarSolicitudItemRequest struct {
	SolicitudItemID string `form:"solicitud_item_id" binding:"required"`
	Fecha           string `form:"fecha" binding:"required"`
	Hora            string `form:"hora" binding:"required"`
	Motivo          string `form:"motivo"`
}
