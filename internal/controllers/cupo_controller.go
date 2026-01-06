package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/appcontext"
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

	cupos, _ := ctrl.service.GetAllByPeriodo(gestion, mes)

	meses := utils.GetMonthNames()
	nombreMes := ""
	if mes >= 1 && mes <= 12 {
		nombreMes = meses[mes]
	}

	users, _ := ctrl.userService.GetByRoleType("SENADOR")

	utils.Render(c, "admin/cupos.html", gin.H{
		"Cupos":     cupos,
		"Users":     users,
		"Gestion":   gestion,
		"Mes":       mes,
		"NombreMes": nombreMes,
		"Meses":     meses,
	})
}

func (ctrl *CupoController) Generar(c *gin.Context) {
	now := time.Now()
	gestionStr := c.PostForm("gestion")
	mesStr := c.PostForm("mes")

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	if gestion == 0 {
		gestion = now.Year()
	}
	if mes == 0 {
		mes = int(now.Month())
	}

	err := ctrl.service.GenerateVouchersForMonth(gestion, mes)
	if err != nil {
		// log.Printf("Error generando cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) Transferir(c *gin.Context) {
	voucherID := c.PostForm("voucher_id")
	destinoID := c.PostForm("destino_id")
	motivo := c.PostForm("motivo")

	gestion := c.PostForm("gestion")
	mes := c.PostForm("mes")

	err := ctrl.service.TransferirVoucher(voucherID, destinoID, motivo)
	if err != nil {
		// log.Printf("Error transfiriendo voucher: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes))
}

func (ctrl *CupoController) Reset(c *gin.Context) {
	now := time.Now()
	gestionStr := c.PostForm("gestion")
	mesStr := c.PostForm("mes")

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	if gestion == 0 {
		gestion = now.Year()
	}
	if mes == 0 {
		mes = int(now.Month())
	}

	err := ctrl.service.ResetVouchersForMonth(gestion, mes)
	if err != nil {
		// log.Printf("Error reset cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}

func (ctrl *CupoController) GetVouchersByCupo(c *gin.Context) {
	cupoID := c.Param("id")

	vouchers, err := ctrl.service.GetVouchersByCupoID(cupoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var suplente *models.Usuario
	if len(vouchers) > 0 {
		senadorID := vouchers[0].SenadorID
		s, err := ctrl.userService.GetSuplenteByTitularID(senadorID)
		if err == nil {
			suplente = s
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"vouchers": vouchers,
		"suplente": suplente,
	})
}

type MonthGroup struct {
	MonthNum  int
	MonthName string
	Vouchers  []models.AsignacionVoucher
}

func (ctrl *CupoController) Derecho(c *gin.Context) {
	id := c.Param("id")
	targetUser, err := ctrl.userService.GetByID(id)
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

	vouchers, _ := ctrl.service.GetVouchersByUsuarioAndGestion(id, gestion)

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

	utils.Render(c, "cupo/derecho.html", gin.H{
		"TargetUser": targetUser,
		"User":       appContextUser,
		"Months":     displayMonths,
		"Gestion":    gestion,
	})
}
