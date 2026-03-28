package controllers

import (
	"net/http"
	"sistema-pasajes/internal/services"
	"strings"

	"github.com/gin-gonic/gin"
)

type CatalogoController struct {
	tipoSolicitudService *services.TipoSolicitudService
	destinoService       *services.DestinoService
	usuarioService       *services.UsuarioService
}

func NewCatalogoController(
	tipoSolicitudService *services.TipoSolicitudService,
	destinoService *services.DestinoService,
	usuarioService *services.UsuarioService,
) *CatalogoController {
	return &CatalogoController{
		tipoSolicitudService: tipoSolicitudService,
		destinoService:       destinoService,
		usuarioService:       usuarioService,
	}
}

func (ctrl *CatalogoController) GetTipos(c *gin.Context) {
	conceptoCodigo := c.Query("concepto_codigo")
	tipos, _ := ctrl.tipoSolicitudService.GetByConcepto(c.Request.Context(), conceptoCodigo)

	c.HTML(http.StatusOK, "catalogos/options_tipos", gin.H{
		"Tipos": tipos,
	})
}

func (ctrl *CatalogoController) GetAmbitos(c *gin.Context) {
	tipoCodigo := c.Query("tipo_solicitud_codigo")
	ambitos, _ := ctrl.tipoSolicitudService.GetAmbitosByTipo(c.Request.Context(), tipoCodigo)

	c.HTML(http.StatusOK, "catalogos/options_ambitos", gin.H{
		"Ambitos": ambitos,
	})
}

func (ctrl *CatalogoController) SearchDestinos(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusOK, []any{})
		return
	}
	destinos, err := ctrl.destinoService.Search(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]map[string]any, len(destinos))
	for i, d := range destinos {
		iata := strings.TrimSpace(strings.ToUpper(d.IATA))
		results[i] = map[string]any{
			"value": iata,
			"label": strings.TrimSpace(d.GetNombreDisplay()),
			"extra": iata,
		}
	}

	c.JSON(http.StatusOK, results)
}

func (ctrl *CatalogoController) SearchStaff(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 3 {
		c.JSON(http.StatusOK, []any{})
		return
	}
	usuarios, err := ctrl.usuarioService.SearchStaff(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	results := make([]map[string]any, len(usuarios))
	for i, u := range usuarios {
		id := strings.TrimSpace(u.ID)
		results[i] = map[string]any{
			"value": id,
			"label": strings.TrimSpace(u.GetNombreCompleto()),
			"extra": strings.TrimSpace(u.CI),
		}
	}

	c.JSON(http.StatusOK, results)
}
