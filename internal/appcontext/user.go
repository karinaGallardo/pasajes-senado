package appcontext

import (
	"context"
	"sistema-pasajes/internal/models"

	"github.com/gin-gonic/gin"
)

type userKey struct{}

func WithUser(ctx context.Context, user *models.Usuario) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}

func FromContext(ctx context.Context) *models.Usuario {
	if u, ok := ctx.Value(userKey{}).(*models.Usuario); ok {
		return u
	}
	return nil
}

func SetUser(c *gin.Context, user *models.Usuario) {
	c.Set("auth_user", user)
	c.Request = c.Request.WithContext(WithUser(c.Request.Context(), user))
}

func CurrentUser(c *gin.Context) *models.Usuario {
	if val, exists := c.Get("auth_user"); exists {
		if u, ok := val.(*models.Usuario); ok {
			return u
		}
	}
	return FromContext(c.Request.Context())
}
