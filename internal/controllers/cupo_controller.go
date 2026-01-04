package controllers

import (
	"fmt"
	"net/http"
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
	// vouchers, _ := ctrl.service.GetAllVouchersByPeriodo(gestion, mes) // Deprecated for main view

	meses := []string{"", "Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio", "Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre"}
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
