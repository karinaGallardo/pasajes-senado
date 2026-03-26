package appcontext

import (
	"context"
	"sistema-pasajes/internal/models"

	"github.com/gin-gonic/gin"
)

type contextKey string

const (
	authUserKey  contextKey = "auth_user"
	userIPKey    contextKey = "user_ip"
	userAgentKey contextKey = "user_agent"
)

func SetUser(c *gin.Context, user *models.Usuario) {
	// Guardar en Gin
	c.Set(string(authUserKey), user)
	// Guardar en el Contexto de la Request (para Servicios)
	ctx := context.WithValue(c.Request.Context(), authUserKey, user)

	// Capturar Metadatos de Red
	ctx = context.WithValue(ctx, userIPKey, c.ClientIP())
	ctx = context.WithValue(ctx, userAgentKey, c.Request.UserAgent())

	c.Request = c.Request.WithContext(ctx)
}

func AuthUser(c *gin.Context) *models.Usuario {
	if val, exists := c.Get(string(authUserKey)); exists {
		if u, ok := val.(*models.Usuario); ok {
			return u
		}
	}
	return nil
}

func GetUserIDFromContext(ctx context.Context) *string {
	if val := ctx.Value(authUserKey); val != nil {
		if u, ok := val.(*models.Usuario); ok {
			return &u.ID
		}
	}
	return nil
}

func GetIPFromContext(ctx context.Context) string {
	if val := ctx.Value(userIPKey); val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func GetUserAgentFromContext(ctx context.Context) string {
	if val := ctx.Value(userAgentKey); val != nil {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}
