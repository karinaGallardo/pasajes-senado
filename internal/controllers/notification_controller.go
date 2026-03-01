package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	service *services.NotificationService
}

func NewNotificationController() *NotificationController {
	return &NotificationController{
		service: services.NewNotificationService(),
	}
}

func (c *NotificationController) GetRecent(ctx *gin.Context) {
	user := appcontext.AuthUser(ctx)
	if user == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	userID := user.ID
	notifications, err := c.service.GetRecentByUserID(ctx, userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	unreadCount, _ := c.service.GetUnreadCount(ctx, userID)

	ctx.JSON(http.StatusOK, gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

func (c *NotificationController) MarkAsRead(ctx *gin.Context) {
	id := ctx.Param("id")
	if err := c.service.MarkAsRead(ctx, id); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Notificación marcada como leída"})
}

func (c *NotificationController) MarkAllAsRead(ctx *gin.Context) {
	user := appcontext.AuthUser(ctx)
	if user == nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Usuario no autenticado"})
		return
	}
	userID := user.ID
	if err := c.service.MarkAllAsRead(ctx, userID); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "Todas las notificaciones marcadas como leídas"})
}
