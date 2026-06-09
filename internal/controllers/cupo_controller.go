package controllers

import (
	"fmt"
	"log"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CupoController struct {
	service     *services.CupoService
	userService *services.UsuarioService
}

func NewCupoController(service *services.CupoService, userService *services.UsuarioService) *CupoController {
	return &CupoController{
		service:     service,
		userService: userService,
	}
}

func (ctrl *CupoController) Index(c *gin.Context) {
	now := time.Now()
	gestionStr := c.DefaultQuery("gestion", strconv.Itoa(now.Year()))
	mesStr := c.DefaultQuery("mes", strconv.Itoa(int(now.Month())))

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	if gestion <= 0 {
		gestion = now.Year()
	}
	if mes <= 0 {
		mes = int(now.Month())
	}

	cupos, _ := ctrl.service.GetAllByPeriodo(c.Request.Context(), gestion, mes)

	meses := utils.GetMonthNames()
	nombreMes := ""
	if mes >= 1 && mes <= 12 {
		nombreMes = meses[mes]
	}

	data := gin.H{
		"Cupos":          cupos,
		"Gestion":        gestion,
		"Mes":            mes,
		"NombreMes":      nombreMes,
		"Meses":          meses,
		"HasCupos":       len(cupos) > 0,
		"CurrentGestion": now.Year(),
		"CurrentMes":     int(now.Month()),
	}

	if c.GetHeader("HX-Request") == "true" {
		utils.Render(c, "admin/components/cupos_response", data)
		return
	}

	users, _ := ctrl.userService.GetByRoleType(c.Request.Context(), "SENADOR")
	data["Users"] = users

	utils.Render(c, "admin/cupos", data)
}

func (ctrl *CupoController) Generar(c *gin.Context) {
	var req dtos.GenerarCupoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cupos")
		return
	}

	now := time.Now()
	gestion, _ := strconv.Atoi(req.Gestion)
	mes, _ := strconv.Atoi(req.Mes)

	if gestion == 0 {
		gestion = now.Year()
	}
	if mes == 0 {
		mes = int(now.Month())
	}

	err := ctrl.service.GenerateCuposDerechoForMonth(c.Request.Context(), gestion, mes)
	if err != nil {
		log.Printf("Error generando cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) TomarCupo(c *gin.Context) {
	var req dtos.TransferirCupoDerechoItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos")
		return
	}

	authUser := appcontext.AuthUser(c)

	if err := ctrl.service.CanTakeCupo(c.Request.Context(), authUser, req.ItemID); err != nil {
		c.Header("HX-Retarget", "#flash-messages")
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	req.TargetUserID = authUser.ID
	req.Motivo = "Tomado por el propio suplente"

	targetUserID := authUser.ID
	err := ctrl.service.TransferirCupoDerecho(c.Request.Context(), req.ItemID, targetUserID, req.Motivo)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	targetURL := fmt.Sprintf("/cupos/derecho/%s/%s/%s", targetUserID, req.Gestion, req.Mes)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}
	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *CupoController) AsignarCupo(c *gin.Context) {
	var req dtos.TransferirCupoDerechoItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	targetUserID := req.TargetUserID
	targetUser, _ := ctrl.userService.GetByID(c.Request.Context(), targetUserID)

	if err := ctrl.service.CanAssignCupo(c.Request.Context(), authUser, targetUser); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	req.Motivo = "Asignación mensual de cupo"

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), req.ItemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	if item.IsVencido() && !authUser.IsAdminOrResponsable() {
		c.String(http.StatusBadRequest, "No se puede asignar un cupo vencido")
		return
	}

	err = ctrl.service.TransferirCupoDerecho(c.Request.Context(), req.ItemID, targetUserID, req.Motivo)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error: "+err.Error())
		return
	}

	targetURL := fmt.Sprintf("/cupos/derecho/%s/%s/%s", targetUserID, req.Gestion, req.Mes)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}
	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *CupoController) Transferir(c *gin.Context) {
	var req dtos.TransferirCupoDerechoItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cupos")
		return
	}

	authUser := appcontext.AuthUser(c)
	if err := ctrl.service.CanTransferCupo(c.Request.Context(), authUser, req.ItemID); err != nil {
		c.String(http.StatusForbidden, err.Error())
		return
	}

	targetUserID := req.TargetUserID
	err := ctrl.service.TransferirCupoDerecho(c.Request.Context(), req.ItemID, targetUserID, req.Motivo)
	if err != nil {
		log.Printf("Error transfiriendo cupo derecho: %v\n", err)
	}

	targetURL := fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", req.Gestion, req.Mes)
	if req.ReturnURL != "" {
		targetURL = req.ReturnURL
	}

	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *CupoController) RevertirTransferencia(c *gin.Context) {
	itemID := c.Param("id")
	gestion := c.Query("gestion")
	mes := c.Query("mes")

	_, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	authUser := appcontext.AuthUser(c)

	if !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permiso para realizar esta acción")
		return
	}

	err = ctrl.service.RevertirTransferencia(c.Request.Context(), itemID)
	if err != nil {
		fmt.Printf("Error revirtiendo transferencia: %v\n", err)
		if c.GetHeader("HX-Request") == "true" {
			c.Header("HX-Trigger", fmt.Sprintf(`{"showAlert": {"icon": "error", "title": "No se puede revertir", "text": "%s"}}`, err.Error()))
			c.Status(http.StatusNoContent)
			return
		}
	}

	targetURL := fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes)
	if ref := c.Request.Referer(); ref != "" {
		targetURL = ref
	}

	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *CupoController) Reset(c *gin.Context) {
	var req dtos.ResetCupoRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cupos")
		return
	}

	now := time.Now()
	gestion, _ := strconv.Atoi(req.Gestion)
	mes, _ := strconv.Atoi(req.Mes)

	if gestion == 0 {
		gestion = now.Year()
	}
	if mes == 0 {
		mes = int(now.Month())
	}

	err := ctrl.service.ResetCuposDerechoForMonth(c.Request.Context(), gestion, mes)
	if err != nil {
		log.Printf("Error reset cupos derecho: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) GetCuposByCupo(c *gin.Context) {
	cupoID := c.Param("id")

	items, err := ctrl.service.GetCuposDerechoByCupoID(c.Request.Context(), cupoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var suplente *models.Usuario
	if len(items) > 0 {
		titularID := items[0].SenTitularID
		s, err := ctrl.userService.GetSuplenteByTitularID(c.Request.Context(), titularID)
		if err == nil {
			suplente = s
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		cupo, _ := ctrl.service.GetByID(c.Request.Context(), cupoID)
		utils.Render(c, "admin/components/modal_derechos_cupo", gin.H{
			"Items":    items,
			"Suplente": suplente,
			"Cupo":     cupo,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"suplente": suplente,
	})
}

func (ctrl *CupoController) GetTransferModal(c *gin.Context) {
	itemID := c.Param("id")
	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	authUser := appcontext.AuthUser(c)
	if !authUser.IsAdminOrResponsable() && authUser.ID != item.SenTitularID {
		c.String(http.StatusForbidden, "No tiene permiso para transferir este derecho")
		return
	}

	senador, _ := ctrl.userService.GetByID(c.Request.Context(), item.SenTitularID)
	suplente, _ := ctrl.userService.GetSuplenteByTitularID(c.Request.Context(), item.SenTitularID)

	var candidates []models.Usuario
	if senador.Tipo == "SENADOR_TITULAR" && suplente != nil {
		candidates = append(candidates, *suplente)
	} else {
		candidates, _ = ctrl.userService.GetByRoleType(c.Request.Context(), "SENADOR")
	}

	gestion := c.Query("gestion")
	mesStr := c.Query("mes")
	mesName := ""
	if mesInt, err := strconv.Atoi(mesStr); err == nil {
		mesName = utils.GetMonthName(mesInt)
	}

	utils.Render(c, "admin/components/modal_transferir_derecho", gin.H{
		"Item":       item,
		"Senador":    senador,
		"Candidates": candidates,
		"Gestion":    gestion,
		"Mes":        mesStr,
		"MesName":    mesName,
		"ReturnURL":  c.Request.Referer(),
	})
}

func (ctrl *CupoController) DerechoByYear(c *gin.Context) {
	senadorUserID := c.Param("senador_user_id")
	gestionStr := c.Param("gestion")

	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), senadorUserID)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	if !strings.Contains(targetUser.Tipo, "SENADOR") {
		c.String(http.StatusBadRequest, "El usuario no es un senador")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var alert string
	if targetUser.OrigenIATA == nil || *targetUser.OrigenIATA == "" {
		alert = "No tiene configurada su ciudad de origen."
	}

	now := time.Now()
	gestion := now.Year()
	if gestionStr != "" {
		if g, err := strconv.Atoi(gestionStr); err == nil {
			gestion = g
		}
	}

	idParaCupos := ctrl.service.ResolveCupoOwner(targetUser)
	items, _ := ctrl.service.GetCuposDerechoByUsuarioAndGestion(c.Request.Context(), idParaCupos, gestion)

	monthGroups := ctrl.service.BuildMonthGroups(items, targetUser, authUser)
	displayMonths := ctrl.service.GetDisplayMonths(monthGroups, gestion)

	utils.Render(c, "cupo/derecho", gin.H{
		"TargetUser":   targetUser,
		"Months":       displayMonths,
		"Gestion":      gestion,
		"AlertaOrigen": alert,
		"OpenTicketID": c.Query("open_ticket_id"),
	})
}

func (ctrl *CupoController) DerechoByMonth(c *gin.Context) {
	senadorUserID := c.Param("senador_user_id")
	gestionStr := c.Param("gestion")
	mesStr := c.Param("mes")

	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), senadorUserID)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	if !strings.Contains(targetUser.Tipo, "SENADOR") {
		c.String(http.StatusBadRequest, "El usuario no es un senador")
		return
	}

	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	var alert string
	if targetUser.OrigenIATA == nil || *targetUser.OrigenIATA == "" {
		alert = "No tiene configurada su ciudad de origen."
	}

	gestion, err := strconv.Atoi(gestionStr)
	if err != nil {
		gestion = time.Now().Year()
	}

	mes, err := strconv.Atoi(mesStr)
	if err != nil {
		mes = 0
	}

	idParaCupos := ctrl.service.ResolveCupoOwner(targetUser)
	items, err := ctrl.service.GetCuposDerechoByUsuario(c.Request.Context(), idParaCupos, gestion, mes)
	if err != nil {
		fmt.Printf("Error obteniendo cupos: %v\n", err)
	}

	monthGroups := ctrl.service.BuildMonthGroups(items, targetUser, authUser)

	var displayMonths []*services.MonthGroup
	for _, mg := range monthGroups {
		if mg != nil && mg.MonthNum == mes {
			displayMonths = append(displayMonths, mg)
		}
	}

	utils.Render(c, "cupo/derecho", gin.H{
		"TargetUser":   targetUser,
		"Months":       displayMonths,
		"Gestion":      gestion,
		"TargetMonth":  mes,
		"AlertaOrigen": alert,
		"OpenTicketID": c.Query("open_ticket_id"),
	})
}
