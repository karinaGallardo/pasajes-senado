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
	c.Set(string(authUserKey), user)
	ctx := context.WithValue(c.Request.Context(), authUserKey, user)
	ctx = ExtractMetadata(c, ctx)
	c.Request = c.Request.WithContext(ctx)
}

func SetMetadata(c *gin.Context) {
	ctx := ExtractMetadata(c, c.Request.Context())
	c.Request = c.Request.WithContext(ctx)
}

func ExtractMetadata(c *gin.Context, ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, userIPKey, c.ClientIP())
	ctx = context.WithValue(ctx, userAgentKey, c.Request.UserAgent())
	return ctx
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
