package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type OpenTicketController struct {
	service          *services.OpenTicketService
	solicitudService *services.SolicitudService
}

func NewOpenTicketController(
	service *services.OpenTicketService,
	solicitudService *services.SolicitudService,
) *OpenTicketController {
	return &OpenTicketController{
		service:          service,
		solicitudService: solicitudService,
	}
}

func (ctrl *OpenTicketController) List(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	title := "Gestión de Pasajes No Utilizados (Open Tickets)"

	tickets, err := ctrl.service.GetScoped(c.Request.Context(), user)
	if err != nil {
		utils.SetErrorMessage(c, "Error al obtener listado de tickets")
		c.Redirect(302, "/dashboard")
		return
	}

	if user.IsSenador() {
		title = "Mis Tramos No Utilizados (Open Tickets)"
	} else if !user.IsAdminOrResponsable() {
		title = "Tramos No Utilizados por Beneficiarios"
	}

	utils.Render(c, "pasajes/open_tickets", gin.H{
		"Title":           title,
		"Tickets":         tickets,
		"ShowBeneficiary": user.IsAdminOrResponsable(),
		"CanManage":       user.IsAdminOrResponsable(),
	})
}

func (ctrl *OpenTicketController) ListByUser(c *gin.Context) {
	ctrl.List(c)
}

func (ctrl *OpenTicketController) GetProgramarModal(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	ticket, err := ctrl.service.GetByID(ctx, id)
	if err != nil {
		c.String(404, "Ticket no encontrado")
		return
	}

	solicitudes, _ := ctrl.solicitudService.GetByUserID(ctx, ticket.UsuarioID, "", "DERECHO")

	utils.Render(c, "pasajes/components/modal_programar_open_ticket", gin.H{
		"Ticket":      ticket,
		"Solicitudes": solicitudes,
	})
}

func (ctrl *OpenTicketController) ProgramarUso(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()
	user := appcontext.AuthUser(c)

	var req struct {
		SolicitudID  string `form:"solicitud_id"`
		MontoCredito string `form:"monto_credito"`
		Obs          string `form:"observaciones"`
	}

	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Header("HX-Refresh", "true")
		c.Status(204)
		return
	}

	if req.SolicitudID != "" {
		if err := ctrl.service.Reserve(ctx, id, req.SolicitudID, user); err != nil {
			utils.SetErrorMessage(c, "Error al programar uso: "+err.Error())
		} else {
			utils.SetSuccessMessage(c, "Uso de ticket programado correctamente")
		}
	} else {
		if err := ctrl.service.RevertToDisponible(ctx, id); err != nil {
			utils.SetErrorMessage(c, "Error al liberar: "+err.Error())
		} else {
			utils.SetSuccessMessage(c, "Ticket liberado correctamente")
		}
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}

func (ctrl *OpenTicketController) Approve(c *gin.Context) {
	id := c.Param("id")
	user := appcontext.AuthUser(c)

	if err := ctrl.service.Approve(c.Request.Context(), id, user); err != nil {
		utils.SetErrorMessage(c, "Error al aprobar: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Open Ticket aprobado — DISPONIBLE")
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}

func (ctrl *OpenTicketController) Release(c *gin.Context) {
	id := c.Param("id")

	if err := ctrl.service.RevertToDisponible(c.Request.Context(), id); err != nil {
		utils.SetErrorMessage(c, "Error al liberar: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Ticket liberado — DISPONIBLE")
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}

func (ctrl *OpenTicketController) Revert(c *gin.Context) {
	id := c.Param("id")

	if err := ctrl.service.RevertToDisponible(c.Request.Context(), id); err != nil {
		utils.SetErrorMessage(c, "Error al revertir: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Ticket revertido — DISPONIBLE")
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}
