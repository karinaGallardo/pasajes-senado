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

func (ctrl *CupoController) Transferir(c *gin.Context) {
	var req dtos.TransferirCupoDerechoItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/cupos")
		return
	}

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), req.ItemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	authUser := appcontext.CurrentUser(c)
	isTitular := authUser.ID == item.SenTitularID
	isSelfAssign := authUser.ID == req.DestinoID && strings.Contains(authUser.Tipo, "SUPLENTE")

	if !authUser.IsAdminOrResponsable() && !isTitular && !isSelfAssign {
		c.String(http.StatusForbidden, "No tiene permiso para transferir este derecho")
		return
	}

	if isSelfAssign {
		gestionInt, _ := strconv.Atoi(req.Gestion)
		mesInt, _ := strconv.Atoi(req.Mes)
		items, _ := ctrl.service.GetCuposDerechoByUsuarioAndGestion(c.Request.Context(), authUser.ID, gestionInt)
		count := 0
		for _, it := range items {
			if it.Mes == mesInt && it.SenAsignadoID == authUser.ID {
				count++
			}
		}
		if count > 0 {
			c.Header("HX-Retarget", "#flash-messages")
			c.String(http.StatusBadRequest, "Límite mensual alcanzado: Solo puede tomar 1 cupo automáticamente.")
			return
		}
	}

	err = ctrl.service.TransferirCupoDerecho(c.Request.Context(), req.ItemID, req.DestinoID, req.Motivo)
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

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), itemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	authUser := appcontext.CurrentUser(c)
	if !authUser.IsAdminOrResponsable() && authUser.ID != item.SenTitularID {
		c.String(http.StatusForbidden, "No tiene permiso para revertir esta transferencia")
		return
	}

	err = ctrl.service.RevertirTransferencia(c.Request.Context(), itemID)
	if err != nil {
		fmt.Printf("Error revirtiendo transferencia: %v\n", err)
	}

	targetURL := fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes)
	if ref := c.Request.Referer(); ref != "" {
		targetURL = ref
	}

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

	authUser := appcontext.CurrentUser(c)
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

type MonthGroup struct {
	MonthNum         int
	MonthName        string
	Items            []models.CupoDerechoItem
	SuplenteHasQuota bool
}

func (ctrl *CupoController) DerechoByYear(c *gin.Context) {
	id := c.Param("id")
	gestionStr := c.Param("gestion")

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
	if gestionStr != "" {
		if g, err := strconv.Atoi(gestionStr); err == nil {
			gestion = g
		}
	}

	var idParaCupos = id
	if targetUser.TitularID != nil {
		idParaCupos = *targetUser.TitularID
	}

	items, _ := ctrl.service.GetCuposDerechoByUsuarioAndGestion(c.Request.Context(), idParaCupos, gestion)

	mesesNames := utils.GetMonthNames()

	grouped := make([]*MonthGroup, 13)
	for i := 1; i <= 12; i++ {
		grouped[i] = &MonthGroup{
			MonthNum:  i,
			MonthName: mesesNames[i],
			Items:     []models.CupoDerechoItem{},
		}
	}

	for _, v := range items {
		if v.Mes >= 1 && v.Mes <= 12 {
			grouped[v.Mes].Items = append(grouped[v.Mes].Items, v)
			if v.SenAsignadoID == id {
				grouped[v.Mes].SuplenteHasQuota = true
			}
		}
	}

	var displayMonths []*MonthGroup
	for i := 1; i <= 12; i++ {
		if len(grouped[i].Items) > 0 {
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

func (ctrl *CupoController) DerechoByMonth(c *gin.Context) {
	id := c.Param("id")
	gestionStr := c.Param("gestion")
	mesStr := c.Param("mes")

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

	gestion, err := strconv.Atoi(gestionStr)
	if err != nil {
		gestion = time.Now().Year() // Fallback safety
	}

	mes, err := strconv.Atoi(mesStr)
	if err != nil {
		mes = 0
	}

	var idParaCupos = id
	if targetUser.TitularID != nil {
		idParaCupos = *targetUser.TitularID
	}

	items, _ := ctrl.service.GetCuposDerechoByUsuarioAndGestion(c.Request.Context(), idParaCupos, gestion)

	mesesNames := utils.GetMonthNames()

	grouped := make([]*MonthGroup, 13)
	for i := 1; i <= 12; i++ {
		grouped[i] = &MonthGroup{
			MonthNum:  i,
			MonthName: mesesNames[i],
			Items:     []models.CupoDerechoItem{},
		}
	}

	for _, v := range items {
		if v.Mes == mes {
			grouped[v.Mes].Items = append(grouped[v.Mes].Items, v)
			if v.SenAsignadoID == id {
				grouped[v.Mes].SuplenteHasQuota = true
			}
		}
	}

	var displayMonths []*MonthGroup
	if mes >= 1 && mes <= 12 {
		displayMonths = append(displayMonths, grouped[mes])
	}

	utils.Render(c, "cupo/derecho", gin.H{
		"TargetUser":  targetUser,
		"User":        appContextUser,
		"Months":      displayMonths,
		"Gestion":     gestion,
		"TargetMonth": mes,
	})
}
