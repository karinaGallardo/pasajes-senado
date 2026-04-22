package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type OpenTicketController struct {
	service       *services.OpenTicketService
	usuarioRepo   *repositories.UsuarioRepository
	solicitudRepo *repositories.SolicitudRepository
}

func NewOpenTicketController(
	service *services.OpenTicketService,
	usuarioRepo *repositories.UsuarioRepository,
	solicitudRepo *repositories.SolicitudRepository,
) *OpenTicketController {
	return &OpenTicketController{
		service:       service,
		usuarioRepo:   usuarioRepo,
		solicitudRepo: solicitudRepo,
	}
}

func (ctrl *OpenTicketController) List(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	var tickets []models.OpenTicket
	var err error
	title := "Gestión de Pasajes No Utilizados (Open Tickets)"

	if user.IsAdminOrResponsable() {
		// Admin/Responsable: Ver todo
		tickets, err = ctrl.service.GetAll(c.Request.Context(), nil)
	} else if user.IsSenador() {
		// Senador: Ver sus propios tickets
		tickets, err = ctrl.service.GetForUser(c.Request.Context(), user.ID)
		title = "Mis Tramos No Utilizados (Open Tickets)"
	} else {
		// Asistente/Encargado: Ver tickets de sus asignados
		managedUsers, _ := ctrl.usuarioRepo.FindByEncargadoID(c.Request.Context(), user.ID)
		if len(managedUsers) > 0 {
			managedIDs := make([]string, 0, len(managedUsers))
			for _, u := range managedUsers {
				managedIDs = append(managedIDs, u.ID)
			}
			// Necesitamos un método en el repo/service que acepte múltiples IDs o filtrar manualmente
			// Por simplicidad, implementaremos GetByUsuarioIDs en el service si es necesario
			// O podemos iterar (no muy eficiente pero para pocos asignados está ok)
			allTickets := []models.OpenTicket{}
			for _, id := range managedIDs {
				tks, _ := ctrl.service.GetForUser(c.Request.Context(), id)
				allTickets = append(allTickets, tks...)
			}
			tickets = allTickets
		}
		title = "Tramos No Utilizados por Beneficiarios"
	}

	if err != nil {
		utils.SetErrorMessage(c, "Error al obtener listado de tickets")
		c.Redirect(302, "/dashboard")
		return
	}

	utils.Render(c, "pasajes/open_tickets", gin.H{
		"Title":           title,
		"Tickets":         tickets,
		"ShowBeneficiary": user.IsAdminOrResponsable() || title == "Tramos No Utilizados por Beneficiarios",
	})
}

func (ctrl *OpenTicketController) ListByUser(c *gin.Context) {
	// Reutilizamos la lógica o redirigimos
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

	// Lógica de visualización para el formulario
	rutaSugerida := ticket.RutaProgramada
	if rutaSugerida == "" {
		rutaSugerida = ticket.TramosNoUsados
	}

	aerolineaSugerida := ticket.AerolineaProgramada
	if aerolineaSugerida == "" {
		if ticket.Pasaje != nil && ticket.Pasaje.Aerolinea != nil {
			aerolineaSugerida = ticket.Pasaje.Aerolinea.Nombre
		}
	}

	utils.Render(c, "pasajes/components/modal_programar_open_ticket", gin.H{
		"Ticket":            ticket,
		"RutaSugerida":      rutaSugerida,
		"AerolineaSugerida": aerolineaSugerida,
	})
}

func (ctrl *OpenTicketController) ProgramarUso(c *gin.Context) {
	id := c.Param("id")
	ctx := c.Request.Context()

	var req struct {
		FechaVuelo string `form:"fecha_vuelo"`
		Ruta       string `form:"ruta"`
		Aerolinea  string `form:"aerolinea"`
		Obs        string `form:"observaciones"`
	}

	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Header("HX-Refresh", "true")
		c.Status(204)
		return
	}

	ticket, err := ctrl.service.GetByID(ctx, id)
	if err != nil {
		c.String(404, "Ticket no encontrado")
		return
	}

	if req.FechaVuelo != "" {
		f, _ := utils.ParseDateTime(req.FechaVuelo)
		ticket.FechaVueloProgramada = f
	}
	ticket.RutaProgramada = req.Ruta
	ticket.AerolineaProgramada = req.Aerolinea
	ticket.Observaciones = req.Obs
	ticket.Estado = models.EstadoOpenTicketReservado

	if err := ctrl.service.Update(ctx, ticket); err != nil {
		utils.SetErrorMessage(c, "Error al programar uso: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Uso de ticket programado correctamente")
	}

	c.Header("HX-Refresh", "true")
	c.Status(204)
}
