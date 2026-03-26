package controllers

import (
	"net/http"
	"sistema-pasajes/internal/appcontext"
	"sistema-pasajes/internal/services"

	"github.com/gin-gonic/gin"
)

type NotificationController struct {
	service     *services.NotificationService
	pushService *services.PushService
}

func NewNotificationController(service *services.NotificationService, pushService *services.PushService) *NotificationController {
	return &NotificationController{
		service:     service,
		pushService: pushService,
	}
}

func (ctrl *NotificationController) GetPendingStats(c *gin.Context) {
	// ... Logic for pending stats (FV-01, FV-05) ...
	// (Si es necesario implementarlo aquí o en otro lugar)
}

func (ctrl *NotificationController) SubscribePush(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var dto services.PushSubscriptionDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.pushService.Subscribe(c.Request.Context(), user.ID, dto); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Suscrito correctamente"})
}

func (ctrl *NotificationController) GetRecent(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	notifs, err := ctrl.service.GetRecentByUserID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	unreadCount, _ := ctrl.service.GetUnreadCount(c.Request.Context(), user.ID)

	c.JSON(http.StatusOK, gin.H{
		"notifications": notifs,
		"unread_count":  unreadCount,
	})
}

func (ctrl *NotificationController) MarkAsRead(c *gin.Context) {
	id := c.Param("id")
	if err := ctrl.service.MarkAsRead(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (ctrl *NotificationController) MarkAllAsRead(c *gin.Context) {
	user := appcontext.AuthUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := ctrl.service.MarkAllAsRead(c.Request.Context(), user.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
