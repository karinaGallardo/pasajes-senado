package controllers

import (
	"fmt"
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/services"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CupoController struct {
	service  *services.CupoService
	userRepo *repositories.UsuarioRepository
}

func NewCupoController() *CupoController {
	db := configs.DB
	return &CupoController{
		service:  services.NewCupoService(db),
		userRepo: repositories.NewUsuarioRepository(db),
	}
}

func (ctrl *CupoController) Index(c *gin.Context) {
	now := time.Now()
	gestionStr := c.DefaultQuery("gestion", strconv.Itoa(now.Year()))
	mesStr := c.DefaultQuery("mes", strconv.Itoa(int(now.Month())))

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	vouchers, _ := ctrl.service.GetAllVouchersByPeriodo(gestion, mes)

	meses := []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
	nombreMes := ""
	if mes >= 1 && mes <= 12 {
		nombreMes = meses[mes]
	}

	users, _ := ctrl.userRepo.FindAll()

	c.HTML(http.StatusOK, "admin/cupos.html", gin.H{
		"User":      c.MustGet("User"),
		"Vouchers":  vouchers,
		"Users":     users,
		"Gestion":   gestion,
		"Mes":       mes,
		"NombreMes": nombreMes,
		"Meses":     meses,
	})
}

func (ctrl *CupoController) Generar(c *gin.Context) {
	gestionStr := c.PostForm("gestion")
	mesStr := c.PostForm("mes")

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	err := ctrl.service.GenerateVouchersForMonth(gestion, mes)
	if err != nil {
		fmt.Printf("Error generando cupos: %v\n", err)
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
		fmt.Printf("Error transfiriendo voucher: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes))
}

func (ctrl *CupoController) Reset(c *gin.Context) {
	gestionStr := c.PostForm("gestion")
	mesStr := c.PostForm("mes")

	gestion, _ := strconv.Atoi(gestionStr)
	mes, _ := strconv.Atoi(mesStr)

	err := ctrl.service.ResetVouchersForMonth(gestion, mes)
	if err != nil {
		fmt.Printf("Error reset cupos: %v\n", err)
	}

	c.Redirect(http.StatusFound, fmt.Sprintf("/admin/cupos?gestion=%d&mes=%d", gestion, mes))
}
