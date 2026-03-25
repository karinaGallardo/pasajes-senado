package controllers

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type LandingController struct{}

func NewLandingController() *LandingController {
	return &LandingController{}
}

func (ctrl *LandingController) ShowAbout(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/about", gin.H{
		"Title": "Acerca del Sistema | SGP-SENADO",
	})
}
