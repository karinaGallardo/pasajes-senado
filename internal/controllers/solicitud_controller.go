package controllers

import (
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/models"
	"sistema-pasajes/internal/services"
	"sistema-pasajes/internal/utils"

	"github.com/gin-gonic/gin"
)

type SolicitudController struct {
	service     *services.SolicitudService
	userService *services.UsuarioService
}

func NewSolicitudController() *SolicitudController {
	return &SolicitudController{
		service:     services.NewSolicitudService(),
		userService: services.NewUsuarioService(),
	}
}

func (ctrl *SolicitudController) Index(c *gin.Context) {
	authUser := appcontext.AuthUser(c)
	if authUser == nil {
		c.Redirect(302, "/auth/login")
		return
	}

	var solicitudes []models.Solicitud
	var err error

	if authUser.IsAdminOrResponsable() {
		solicitudes, err = ctrl.service.GetAll(c.Request.Context())
	} else {
		solicitudes, err = ctrl.service.GetByUserIdOrAccesibleByEncargadoID(c.Request.Context(), authUser.ID)
	}

	if err != nil {
		solicitudes = []models.Solicitud{}
	}

	userIDsMap := make(map[string]bool)
	for _, s := range solicitudes {
		if s.CreatedBy != nil {
			userIDsMap[*s.CreatedBy] = true
		}
		if s.UpdatedBy != nil {
			userIDsMap[*s.UpdatedBy] = true
		}
	}

	var ids []string
	for id := range userIDsMap {
		ids = append(ids, id)
	}

	usuariosMap := make(map[string]*models.Usuario)
	if len(ids) > 0 {
		usuarios, _ := ctrl.userService.GetByIDs(c.Request.Context(), ids)
		for i := range usuarios {
			usuariosMap[usuarios[i].ID] = &usuarios[i]
		}
	}

	utils.Render(c, "solicitud/index", gin.H{
		"Title":       "Bandeja de Solicitudes",
		"Solicitudes": solicitudes,
		"Usuarios":    usuariosMap,
	})
}

func (ctrl *SolicitudController) Create(c *gin.Context)    {}
func (ctrl *SolicitudController) Store(c *gin.Context)     {}
func (ctrl *SolicitudController) Show(c *gin.Context)      {}
func (ctrl *SolicitudController) Edit(c *gin.Context)      {}
func (ctrl *SolicitudController) Update(c *gin.Context)    {}
func (ctrl *SolicitudController) CheckCupo(c *gin.Context) {}
func (ctrl *SolicitudController) Approve(c *gin.Context)   {}
func (ctrl *SolicitudController) Reject(c *gin.Context)    {}
func (ctrl *SolicitudController) PrintPV01(c *gin.Context) {}
