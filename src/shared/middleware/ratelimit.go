package middleware

import (
	"strings"
	"time"

	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// ServiceSubjectPrefix marca los tokens que taxi-back-go firma para sus propios
// pedidos (p. ej. "service:taxi-back-go"). A los clientes nunca les entrega uno.
const ServiceSubjectPrefix = "service:"

// RateLimit acepta maxPerMinute peticiones por minuto de cada sesion de mapas.
// La clave lleva tambien la IP porque un enlace compartido tiene una sola
// sesion para todos sus espectadores. Va despues de Auth.
//
// Los pedidos de taxi-back-go no se limitan: concentra los de todas las
// empresas en una sola sesion y ya tiene sus propios limites.
func RateLimit(name string, maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Next: func(c *fiber.Ctx) bool {
			subject, _ := c.Locals(SubjectLocalKey).(string)
			return strings.HasPrefix(subject, ServiceSubjectPrefix)
		},
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			subject, _ := c.Locals(SubjectLocalKey).(string)
			return "ratelimit:" + name + ":" + subject + ":" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, "too many requests, please try again later", fiber.StatusTooManyRequests)
		},
	})
}

// HealthRateLimit protege el endpoint publico de liveness.
func HealthRateLimit() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        120,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:health:" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusTooManyRequests)
		},
	})
}
