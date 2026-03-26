package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter maneja limitadores por dirección IP.
type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

// NewIPRateLimiter crea un nuevo limitador de tasa basado en IP.
// r: peticiones por segundo.
// b: ráfaga (burst) permitida.
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

// GetLimiter obtiene el limitador para una IP específica.
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware es el middleware para Gin.
func RateLimitMiddleware(limiter *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		l := limiter.GetLimiter(ip)
		if !l.Allow() {
			c.HTML(http.StatusTooManyRequests, "errors/429", gin.H{
				"Title":   "Demasiadas peticiones",
				"Message": "Has excedido el límite de intentos permitidos. Por favor, espera un momento antes de volver a intentar.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// CleanupIPLimiters limpia limitadores antiguos periódicamente.
// (Opcional, para evitar consumo infinito de memoria en servidores de larga duración)
func (i *IPRateLimiter) CleanupIPLimiters() {
	for {
		time.Sleep(time.Hour)
		i.mu.Lock()
		// Reiniciar el mapa si crece demasiado (estrategia simple)
		if len(i.ips) > 10000 {
			i.ips = make(map[string]*rate.Limiter)
		}
		i.mu.Unlock()
	}
}
