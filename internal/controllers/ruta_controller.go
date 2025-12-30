package controllers

import (
	"log"
	"net/http"
	"sistema-pasajes/internal/configs"
	"sistema-pasajes/internal/dtos"
	"sistema-pasajes/internal/services"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RutaController struct {
	rutaService      *services.RutaService
	aerolineaService *services.AerolineaService
}

func NewRutaController() *RutaController {
	db := configs.DB
	return &RutaController{
		rutaService:      services.NewRutaService(db),
		aerolineaService: services.NewAerolineaService(db),
	}
}

func (ctrl *RutaController) Index(c *gin.Context) {
	rutas, _ := ctrl.rutaService.GetAll()
	aerolineas, _ := ctrl.aerolineaService.GetAll()

	c.HTML(http.StatusOK, "admin/rutas.html", gin.H{
		"User":       c.MustGet("User"),
		"Rutas":      rutas,
		"Aerolineas": aerolineas,
		"Title":      "Gestión de Rutas y Tarifas",
	})
}

func (ctrl *RutaController) Store(c *gin.Context) {
	var req dtos.CreateRutaRequest
	if err := c.ShouldBind(&req); err != nil {
		c.Redirect(http.StatusFound, "/admin/rutas?error=DatosInvalidos")
		return
	}

	_, err := ctrl.rutaService.Create(req.Tramo, req.Sigla, req.Ambito)
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

	err = ctrl.rutaService.AssignContract(req.RutaID, req.AerolineaID, monto)
	if err != nil {
		log.Printf("Error adding contract: %v", err)
	}
	c.Redirect(http.StatusFound, "/admin/rutas")
}
