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
	service       *services.CupoService
	userService   *services.UsuarioService
	reportService *services.ReportService
}

func NewCupoController() *CupoController {
	return &CupoController{
		service:       services.NewCupoService(),
		userService:   services.NewUsuarioService(),
		reportService: services.NewReportService(),
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
	if !strings.Contains(authUser.Tipo, "SUPLENTE") {
		c.String(http.StatusForbidden, "Solo los senadores suplentes pueden tomar cupos")
		return
	}

	// For security, ensure the destination is the authenticated user
	req.TargetUserID = authUser.ID
	req.Motivo = "Tomado por el propio suplente"

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), req.ItemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	if item.FechaHasta != nil && time.Now().After(item.FechaHasta.AddDate(0, 0, 1)) {
		c.String(http.StatusBadRequest, "No se puede tomar un cupo vencido")
		return
	}

	// Monthly Check
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

	targetUserID := authUser.ID
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

func (ctrl *CupoController) AsignarCupo(c *gin.Context) {
	var req dtos.TransferirCupoDerechoItemRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, "Datos inválidos")
		return
	}

	authUser := appcontext.AuthUser(c)
	targetUserID := req.TargetUserID
	isEncargado := false
	targetUser, _ := ctrl.userService.GetByID(c.Request.Context(), targetUserID)
	if targetUser != nil && targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID {
		isEncargado = true
	}

	if !authUser.IsAdminOrResponsable() && !isEncargado {
		c.String(http.StatusForbidden, "No tiene permiso para asignar cupos")
		return
	}

	req.Motivo = "Asignación mensual de cupo"

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), req.ItemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	if item.FechaHasta != nil && time.Now().After(item.FechaHasta.AddDate(0, 0, 1)) {
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
	if !authUser.IsAdminOrResponsable() {
		c.String(http.StatusForbidden, "No tiene permiso para transferir derechos")
		return
	}

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), req.ItemID)
	if err != nil {
		c.String(http.StatusNotFound, "Derecho no encontrado")
		return
	}

	if item.FechaHasta != nil && time.Now().After(item.FechaHasta.AddDate(0, 0, 1)) {
		c.String(http.StatusBadRequest, "No se puede transferir un cupo vencido")
		return
	}
	targetUserID := req.TargetUserID
	err = ctrl.service.TransferirCupoDerecho(c.Request.Context(), req.ItemID, targetUserID, req.Motivo)
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
	}

	targetURL := fmt.Sprintf("/admin/cupos?gestion=%s&mes=%s", gestion, mes)
	if ref := c.Request.Referer(); ref != "" {
		targetURL = ref
	}

	c.Redirect(http.StatusFound, targetURL)
}

func (ctrl *CupoController) PrintRequests(c *gin.Context) {
	id := c.Param("id")

	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Cupo no encontrado")
		return
	}

	pdfBytes, err := ctrl.reportService.GenerateCupoSolicitudesPDF(c.Request.Context(), item.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error generando reporte: "+err.Error())
		return
	}

	filename := fmt.Sprintf("solicitudes_cupo_%s.pdf", item.Semana)
	c.Header("Content-Disposition", "inline; filename="+filename)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

func (ctrl *CupoController) GetPrintModal(c *gin.Context) {
	id := c.Param("id")
	item, err := ctrl.service.GetCupoDerechoItemByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Cupo no encontrado")
		return
	}

	utils.Render(c, "cupo/modal_print", gin.H{
		"Item": item,
	})
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

type Permissions struct {
	CanManage          bool
	CanTransfer        bool
	CanRevert          bool
	CanPrint           bool
	CanTomarCupo       bool
	CanAsignarCupo     bool
	CanCreateIda       bool
	CanCreateVuelta    bool
	CanCreateIdaVuelta bool
	CanEditIda         bool
	CanEditVuelta      bool
	CanViewIda         bool
	CanViewVuelta      bool
	CanDescargo        bool
}

type CupoDerechoItemView struct {
	models.CupoDerechoItem
	Permissions Permissions
}

type MonthGroup struct {
	MonthNum         int
	MonthName        string
	Items            []CupoDerechoItemView
	SuplenteHasQuota bool
	TargetHasQuota   bool
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

	now := time.Now()
	gestion := now.Year()
	if gestionStr != "" {
		if g, err := strconv.Atoi(gestionStr); err == nil {
			gestion = g
		}
	}

	var idParaCupos = senadorUserID
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
			Items:     []CupoDerechoItemView{},
		}
	}

	for _, v := range items {
		if v.Mes >= 1 && v.Mes <= 12 {
			grouped[v.Mes].Items = append(grouped[v.Mes].Items, CupoDerechoItemView{CupoDerechoItem: v})
			if strings.Contains(targetUser.Tipo, "SUPLENTE") && v.SenAsignadoID == senadorUserID {
				grouped[v.Mes].SuplenteHasQuota = true
			}
			if v.SenAsignadoID == senadorUserID {
				grouped[v.Mes].TargetHasQuota = true
			}
		}
	}

	isViewerAdminOrResponsable := authUser.IsAdminOrResponsable()
	isViewerSuplente := strings.Contains(authUser.Tipo, "SUPLENTE")
	isTargetSuplente := strings.Contains(targetUser.Tipo, "SUPLENTE")
	isEncargado := targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID

	for i := 1; i <= 12; i++ {
		// Populate Permissions for each item
		for j := range grouped[i].Items {
			item := &grouped[i].Items[j]
			perms := Permissions{}
			modelItem := item.CupoDerechoItem

			isDisponible := modelItem.IsDisponible()
			isVencido := modelItem.IsVencido()
			isTransferido := modelItem.EsTransferido
			isOwner := modelItem.SenAsignadoID == targetUser.ID
			targetHasQuota := grouped[i].TargetHasQuota

			// 1. Admin Actions
			if isViewerAdminOrResponsable {
				if isTransferido {
					perms.CanRevert = true
				}
				if modelItem.GetSolicitudIda() != nil || modelItem.GetSolicitudVuelta() != nil {
					perms.CanPrint = true
				}
			}

			// 2. Tomar / Asignar Cupo (Para el Target Suplente)
			hasTitular := targetUser.TitularID != nil
			if isDisponible && !isVencido && hasTitular && modelItem.SenAsignadoID == *targetUser.TitularID {
				// Opción 1: El mismo suplente toma su cupo (Solo si no tiene cupo)
				if isTargetSuplente && !targetHasQuota && isViewerSuplente && authUser.ID == targetUser.ID {
					perms.CanTomarCupo = true
				}

				// Opción 2: Encargado asigna (Solo si no tiene cupo)
				if isEncargado && !targetHasQuota && isTargetSuplente {
					perms.CanAsignarCupo = true
				}

				// Opción 3: Admin/Responsable asigna (Solo si no tiene cupo)
				if isViewerAdminOrResponsable && !targetHasQuota && isTargetSuplente && !isTransferido {
					perms.CanAsignarCupo = true
				}
			}

			// 3. Transferencia (Admin/Responsable)
			if isViewerAdminOrResponsable && isDisponible && !isVencido && !perms.CanAsignarCupo && !isTransferido {
				perms.CanTransfer = true
			}

			// 3. Solicitudes (Owner Only)
			if isOwner {
				solIda := modelItem.GetSolicitudIda()
				solVuelta := modelItem.GetSolicitudVuelta()

				// Ida
				if solIda == nil {
					if !isVencido {
						perms.CanCreateIda = true
					}
				} else {
					if solIda.GetEstado() == "SOLICITADO" {
						perms.CanEditIda = true
					} else {
						perms.CanViewIda = true
					}
				}

				// Vuelta
				if solVuelta == nil {
					if !isVencido && solIda != nil {
						perms.CanCreateVuelta = true
					}
				} else {
					if solVuelta.GetEstado() == "SOLICITADO" {
						perms.CanEditVuelta = true
					} else {
						perms.CanViewVuelta = true
					}
				}

				// Ida y Vuelta (Round Trip) in Single Request
				// Only possible if NO requests exist yet and cupo is active
				if solIda == nil && solVuelta == nil && !isVencido {
					perms.CanCreateIdaVuelta = true
				}

				// Descargo
				perms.CanDescargo = true
			}

			item.Permissions = perms
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
		"Months":     displayMonths,
		"Gestion":    gestion,
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

	gestion, err := strconv.Atoi(gestionStr)
	if err != nil {
		gestion = time.Now().Year()
	}

	mes, err := strconv.Atoi(mesStr)
	if err != nil {
		mes = 0
	}

	var idParaCupos = senadorUserID
	if targetUser.TitularID != nil {
		idParaCupos = *targetUser.TitularID
	}

	items, err := ctrl.service.GetCuposDerechoByUsuario(c.Request.Context(), idParaCupos, gestion, mes)
	if err != nil {
		fmt.Printf("Error obteniendo cupos: %v\n", err)
	}

	mesName := utils.GetMonthName(mes)

	var viewItems []CupoDerechoItemView
	for _, v := range items {
		viewItems = append(viewItems, CupoDerechoItemView{CupoDerechoItem: v})
	}

	currentMonthGroup := &MonthGroup{
		MonthNum:  mes,
		MonthName: mesName,
		Items:     viewItems,
	}

	for _, v := range items {
		if strings.Contains(targetUser.Tipo, "SUPLENTE") && v.SenAsignadoID == senadorUserID {
			currentMonthGroup.SuplenteHasQuota = true
		}
		if v.SenAsignadoID == senadorUserID {
			currentMonthGroup.TargetHasQuota = true
		}
	}

	isViewerAdminOrResponsable := authUser.IsAdminOrResponsable()
	isViewerSuplente := strings.Contains(authUser.Tipo, "SUPLENTE")
	isTargetSuplente := strings.Contains(targetUser.Tipo, "SUPLENTE")
	isEncargado := targetUser.EncargadoID != nil && *targetUser.EncargadoID == authUser.ID

	// Populate Permissions
	for j := range currentMonthGroup.Items {
		item := &currentMonthGroup.Items[j]
		perms := Permissions{}
		modelItem := item.CupoDerechoItem

		isDisponible := modelItem.IsDisponible()
		isVencido := modelItem.IsVencido()
		isTransferido := modelItem.EsTransferido
		isOwner := modelItem.SenAsignadoID == targetUser.ID
		targetHasQuota := currentMonthGroup.TargetHasQuota

		// 1. Admin Actions (Calculo base)
		if isViewerAdminOrResponsable {
			if isTransferido {
				perms.CanRevert = true
			}
			if modelItem.GetSolicitudIda() != nil || modelItem.GetSolicitudVuelta() != nil {
				perms.CanPrint = true
			}
		}

		// 2. Tomar / Asignar Cupo (Para el Target Suplente)
		hasTitular := targetUser.TitularID != nil
		if isDisponible && !isVencido && hasTitular && modelItem.SenAsignadoID == *targetUser.TitularID {
			// Opción 1: El mismo suplente toma su cupo (Solo si no tiene cupo)
			if isTargetSuplente && !targetHasQuota && isViewerSuplente && authUser.ID == targetUser.ID {
				perms.CanTomarCupo = true
			}

			// Opción 2: Encargado asigna (Solo si no tiene cupo)
			if isEncargado && !targetHasQuota && isTargetSuplente {
				perms.CanAsignarCupo = true
			}

			// Opción 3: Admin/Responsable asigna (Solo si no tiene cupo)
			if isViewerAdminOrResponsable && !targetHasQuota && isTargetSuplente && !isTransferido {
				perms.CanAsignarCupo = true
			}
		}

		// 3. Transferencia (Admin/Responsable)
		// Si es Admin y NO se activó la asignación directa (porque ya tiene cupo o falta idoneidad), mostramos Transferir
		if isViewerAdminOrResponsable && isDisponible && !isVencido && !perms.CanAsignarCupo {
			perms.CanTransfer = true
		}

		// 3. Solicitudes (Owner Only)
		if isOwner {
			solIda := modelItem.GetSolicitudIda()
			solVuelta := modelItem.GetSolicitudVuelta()

			// Ida
			if solIda == nil {
				if !isVencido {
					perms.CanCreateIda = true
				}
			} else {
				if solIda.GetEstado() == "SOLICITADO" {
					perms.CanEditIda = true
				} else {
					perms.CanViewIda = true
				}
			}

			// Vuelta
			if solVuelta == nil {
				if !isVencido {
					perms.CanCreateVuelta = true
				}
			} else {
				if solVuelta.GetEstado() == "SOLICITADO" {
					perms.CanEditVuelta = true
				} else {
					perms.CanViewVuelta = true
				}
			}

			// Ida y Vuelta (Round Trip) in Single Request
			if solIda == nil && solVuelta == nil && !isVencido {
				perms.CanCreateIdaVuelta = true
			}

			// Descargo
			perms.CanDescargo = true
		}

		item.Permissions = perms
	}

	displayMonths := []*MonthGroup{currentMonthGroup}

	utils.Render(c, "cupo/derecho", gin.H{
		"TargetUser":  targetUser,
		"Months":      displayMonths,
		"Gestion":     gestion,
		"TargetMonth": mes,
	})
}
