package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RutaController struct {
	rutaService      *services.RutaService
	aerolineaService *services.AerolineaService
	destinoService   *services.DestinoService
}

func NewRutaController(
	rutaService *services.RutaService,
	aerolineaService *services.AerolineaService,
	destinoService *services.DestinoService,
) *RutaController {
	return &RutaController{
		rutaService:      rutaService,
		aerolineaService: aerolineaService,
		destinoService:   destinoService,
	}
}

func (ctrl *RutaController) Index(c *gin.Context) {
	rutas, _ := ctrl.rutaService.GetAll(c.Request.Context())
	aerolineas, _ := ctrl.aerolineaService.GetAll(c.Request.Context())

	utils.Render(c, "admin/rutas", gin.H{
		"Rutas":      rutas,
		"Aerolineas": aerolineas,
		"Title":      "Gestión de Rutas y Tarifas",
	})
}

func (ctrl *RutaController) Search(c *gin.Context) {
	query := c.Query("q")
	if len(query) < 3 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}

	rutas, err := ctrl.rutaService.Search(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type searchResult struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}

	results := make([]searchResult, 0, len(rutas))
	for _, r := range rutas {
		results = append(results, searchResult{
			Value: r.ID,
			Label: r.GetTramoDisplay(),
		})
	}

	c.JSON(http.StatusOK, results)
}

func (ctrl *RutaController) Store(c *gin.Context) {
	var req dtos.CreateRutaRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/rutas?error=DatosInvalidos")
		return
	}

	_, err := ctrl.rutaService.Create(c.Request.Context(), req.OrigenIATA, req.EscalasIATA, req.DestinoIATA)
	if err != nil {
		log.Printf("Error creando ruta: %v", err)
	}
	c.Redirect(http.StatusFound, "/admin/rutas")
}

func (ctrl *RutaController) AddContract(c *gin.Context) {
	var req dtos.AddContractRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/rutas?error=DatosContratoInvalidos")
		return
	}

	monto, err := strconv.ParseFloat(req.Monto, 64)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/rutas?error=MontoInvalido")
		return
	}

	err = ctrl.rutaService.AssignContract(c.Request.Context(), req.RutaID, req.AerolineaID, monto)
	if err != nil {
		log.Printf("Error adding contract: %v", err)
	}

	// For HTMX requests, we might want to refresh the table.
	// But the Redirect works too.
	c.Redirect(http.StatusFound, "/admin/rutas")
}

func (ctrl *RutaController) DeleteContract(c *gin.Context) {
	contractID := c.Param("id")
	err := ctrl.rutaService.RemoveContract(c.Request.Context(), contractID)
	if err != nil {
		log.Printf("Error deleting contract: %v", err)
	}
	c.Redirect(http.StatusFound, "/admin/rutas")
}

func (ctrl *RutaController) GetContractModal(c *gin.Context) {
	rutaID := c.Query("ruta_id")
	ruta, err := ctrl.rutaService.GetByID(c.Request.Context(), rutaID)
	if err != nil {
		c.String(http.StatusNotFound, "Ruta no encontrada")
		return
	}

	aerolineas, _ := ctrl.aerolineaService.GetAll(c.Request.Context())

	utils.Render(c, "admin/components/modal_tarifa_ruta", gin.H{
		"Ruta":       ruta,
		"Aerolineas": aerolineas,
	})
}
