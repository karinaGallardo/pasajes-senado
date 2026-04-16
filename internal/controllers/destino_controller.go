package controllers

import (
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/repositories"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type DestinoController struct {
	service    *services.DestinoService
	ambitoRepo *repositories.AmbitoViajeRepository
	deptoRepo  *repositories.DepartamentoRepository
}

func NewDestinoController(
	service *services.DestinoService,
	ambitoRepo *repositories.AmbitoViajeRepository,
	deptoRepo *repositories.DepartamentoRepository,
) *DestinoController {
	return &DestinoController{
		service:    service,
		ambitoRepo: ambitoRepo,
		deptoRepo:  deptoRepo,
	}
}

func capitalize(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

func (ctrl *DestinoController) Index(c *gin.Context) {
	ctx := c.Request.Context()
	pageSize := 10
	destinos, total, err := ctrl.service.Search(ctx, "", "", 1, pageSize)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cargar destinos")
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	utils.Render(c, "admin/destinos/index", gin.H{
		"Destinos":   destinos,
		"Title":      "Gestión de Aeropuertos/Destinos",
		"Page":       1,
		"Total":      int(total),
		"TotalPages": totalPages,
		"PageSize":   pageSize,
		"IsHTMX":     c.GetHeader("HX-Request") != "",
	})
}

func (ctrl *DestinoController) Table(c *gin.Context) {
	ctx := c.Request.Context()
	query := c.Query("q")
	ambito := c.Query("ambito")

	page := 1
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}

	pageSize := 10

	destinos, total, err := ctrl.service.Search(ctx, query, ambito, page, pageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))

	utils.Render(c, "admin/destinos/table", gin.H{
		"Destinos":   destinos,
		"Page":       page,
		"Total":      int(total),
		"TotalPages": totalPages,
		"PageSize":   pageSize,
		"Query":      query,
		"IsHTMX":     c.GetHeader("HX-Request") != "",
	})
}

func (ctrl *DestinoController) New(c *gin.Context) {
	ctx := c.Request.Context()
	ambitos, _ := ctrl.ambitoRepo.FindAll(ctx)
	deptos, _ := ctrl.deptoRepo.FindAll(ctx)

	utils.Render(c, "admin/destinos/form_modal", gin.H{
		"Title":   "Nuevo Destino",
		"Ambitos": ambitos,
		"Deptos":  deptos,
	})
}

func (ctrl *DestinoController) Store(c *gin.Context) {
	var req dtos.CreateDestinoRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos: "+err.Error())
		c.Redirect(http.StatusFound, "/admin/destinos")
		return
	}

	var deptoCode *string
	if req.DepartamentoCodigo != "" {
		deptoCode = &req.DepartamentoCodigo
	}

	var pais *string
	if req.Pais != "" {
		p := strings.ToUpper(strings.TrimSpace(req.Pais))
		pais = &p
	}

	model := models.Destino{
		IATA:               strings.ToUpper(strings.TrimSpace(req.IATA)),
		Ciudad:             capitalize(req.Ciudad),
		Aeropuerto:         capitalize(req.Aeropuerto),
		AmbitoCodigo:       strings.ToUpper(req.AmbitoCodigo),
		DepartamentoCodigo: deptoCode,
		Pais:               pais,
		Estado:             req.Estado == "on",
	}

	if err := ctrl.service.Create(c.Request.Context(), &model); err != nil {
		utils.SetErrorMessage(c, "Error al crear destino: "+err.Error())
	} else {
		utils.SetSuccessMessage(c, "Destino creado correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/destinos")
}

func (ctrl *DestinoController) Edit(c *gin.Context) {
	iata := c.Param("id")
	ctx := c.Request.Context()
	destino, err := ctrl.service.GetByIATA(ctx, iata)
	if err != nil {
		c.String(http.StatusNotFound, "Destino no encontrado")
		return
	}

	ambitos, _ := ctrl.ambitoRepo.FindAll(ctx)
	deptos, _ := ctrl.deptoRepo.FindAll(ctx)

	utils.Render(c, "admin/destinos/form_modal", gin.H{
		"Destino": destino,
		"Title":   "Editar Destino",
		"Ambitos": ambitos,
		"Deptos":  deptos,
	})
}

func (ctrl *DestinoController) Update(c *gin.Context) {
	iata := c.Param("id")
	var req dtos.UpdateDestinoRequest
	if err := c.ShouldBind(&req); err != nil {
		utils.SetErrorMessage(c, "Datos inválidos")
		c.Redirect(http.StatusFound, "/admin/destinos")
		return
	}

	existing, err := ctrl.service.GetByIATA(c.Request.Context(), iata)
	if err != nil {
		utils.SetErrorMessage(c, "Destino no encontrado")
		c.Redirect(http.StatusFound, "/admin/destinos")
		return
	}

	var deptoCode *string
	if req.DepartamentoCodigo != "" {
		deptoCode = &req.DepartamentoCodigo
	}

	var pais *string
	if req.Pais != "" {
		p := strings.ToUpper(strings.TrimSpace(req.Pais))
		pais = &p
	}

	existing.Ciudad = capitalize(req.Ciudad)
	existing.Aeropuerto = capitalize(req.Aeropuerto)
	existing.AmbitoCodigo = strings.ToUpper(req.AmbitoCodigo)
	existing.DepartamentoCodigo = deptoCode
	existing.Pais = pais
	existing.Estado = req.Estado == "on"

	if err := ctrl.service.Update(c.Request.Context(), existing); err != nil {
		utils.SetErrorMessage(c, "Error al actualizar destino")
	} else {
		utils.SetSuccessMessage(c, "Destino actualizado correctamente")
	}

	c.Redirect(http.StatusFound, "/admin/destinos")
}

func (ctrl *DestinoController) Toggle(c *gin.Context) {
	iata := c.Param("id")
	err := ctrl.service.Toggle(c.Request.Context(), iata)
	if err != nil {
		utils.SetErrorMessage(c, "Error al cambiar estado")
	} else {
		utils.SetSuccessMessage(c, "Estado actualizado correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/destinos")
}

func (ctrl *DestinoController) Delete(c *gin.Context) {
	iata := c.Param("id")
	err := ctrl.service.Delete(c.Request.Context(), iata)
	if err != nil {
		utils.SetErrorMessage(c, "Error al eliminar destino")
	} else {
		utils.SetSuccessMessage(c, "Destino eliminado correctamente")
	}
	c.Redirect(http.StatusFound, "/admin/destinos")
}
