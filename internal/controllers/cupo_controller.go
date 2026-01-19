package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CupoController struct {
	service     *services.CupoService
	userService *services.UsuarioService
}

func NewCupoController() *CupoController {
	return &CupoController{
		service:     services.NewCupoService(),
		userService: services.NewUsuarioService(),
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
		"HasVouchers":    len(cupos) > 0,
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

	err := ctrl.service.GenerateVouchersForMonth(c.Request.Context(), gestion, mes)
	if err != nil {
		// log.Printf("Error generando cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) Transferir(c *gin.Context) {
	var req dtos.TransferirVoucherRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cupos")
		return
	}

	err := ctrl.service.TransferirVoucher(c.Request.Context(), req.VoucherID, req.DestinoID, req.Motivo)
	if err != nil {
		// log.Printf("Error transfiriendo voucher: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", req.Gestion, req.Mes))
}

func (ctrl *CupoController) RevertirTransferencia(c *gin.Context) {
	voucherID := c.Param("id")
	gestion := c.Query("gestion")
	mes := c.Query("mes")

	err := ctrl.service.RevertirTransferencia(c.Request.Context(), voucherID)
	if err != nil {
		fmt.Printf("Error revirtiendo transferencia: %v\n", err)
	}

	targetURL := fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes)
	if c.GetHeader("HX-Request") == "true" {
		c.Header("HX-Redirect", targetURL)
		c.Status(http.StatusOK)
		return
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

	err := ctrl.service.ResetVouchersForMonth(c.Request.Context(), gestion, mes)
	if err != nil {
		// log.Printf("Error reset cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) GetVouchersByCupo(c *gin.Context) {
	cupoID := c.Param("id")

	vouchers, err := ctrl.service.GetVouchersByCupoID(c.Request.Context(), cupoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var suplente *models.Usuario
	if len(vouchers) > 0 {
		senadorID := vouchers[0].SenadorID
		s, err := ctrl.userService.GetSuplenteByTitularID(c.Request.Context(), senadorID)
		if err == nil {
			suplente = s
		}
	}

	if c.GetHeader("HX-Request") == "true" {
		cupo, _ := ctrl.service.GetByID(c.Request.Context(), cupoID)
		utils.Render(c, "admin/components/modal_vouchers_cupo", gin.H{
			"Vouchers": vouchers,
			"Suplente": suplente,
			"Cupo":     cupo,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vouchers": vouchers,
		"suplente": suplente,
	})
}

func (ctrl *CupoController) GetTransferModal(c *gin.Context) {
	voucherID := c.Param("id")
	voucher, err := ctrl.service.GetVoucherByID(c.Request.Context(), voucherID)
	if err != nil {
		c.String(http.StatusNotFound, "Voucher no encontrado")
		return
	}

	senador, _ := ctrl.userService.GetByID(c.Request.Context(), voucher.SenadorID)
	suplente, _ := ctrl.userService.GetSuplenteByTitularID(c.Request.Context(), voucher.SenadorID)

	var candidates []models.Usuario
	if senador.Tipo == "SENADOR_TITULAR" && suplente != nil {
		candidates = append(candidates, *suplente)
	} else {
		candidates, _ = ctrl.userService.GetByRoleType(c.Request.Context(), "SENADOR")
	}

	utils.Render(c, "admin/components/modal_transferir_voucher", gin.H{
		"Voucher":    voucher,
		"Senador":    senador,
		"Candidates": candidates,
		"Gestion":    c.Query("gestion"),
		"Mes":        c.Query("mes"),
	})
}

type MonthGroup struct {
	MonthNum  int
	MonthName string
	Vouchers  []models.AsignacionVoucher
}

func (ctrl *CupoController) Derecho(c *gin.Context) {
	id := c.Param("id")
	targetUser, err := ctrl.userService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Usuario no encontrado")
		return
	}

	appContextUser := appcontext.CurrentUser(c)
	if appContextUser == nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	now := time.Now()
	gestion := now.Year()

	vouchers, _ := ctrl.service.GetVouchersByUsuarioAndGestion(c.Request.Context(), id, gestion)

	mesesNames := utils.GetMonthNames()

	grouped := make([]*MonthGroup, 13)
	for i := 1; i <= 12; i++ {
		grouped[i] = &MonthGroup{
			MonthNum:  i,
			MonthName: mesesNames[i],
			Vouchers:  []models.AsignacionVoucher{},
		}
	}

	for _, v := range vouchers {
		if v.Mes >= 1 && v.Mes <= 12 {
			grouped[v.Mes].Vouchers = append(grouped[v.Mes].Vouchers, v)
		}
	}

	var displayMonths []*MonthGroup
	for i := 1; i <= 12; i++ {
		if len(grouped[i].Vouchers) > 0 {
			displayMonths = append(displayMonths, grouped[i])
		}
	}

	utils.Render(c, "cupo/derecho", gin.H{
		"TargetUser": targetUser,
		"User":       appContextUser,
		"Months":     displayMonths,
		"Gestion":    gestion,
	})
}
