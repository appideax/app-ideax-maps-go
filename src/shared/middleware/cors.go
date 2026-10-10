package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// CORS responde con el origen que llama (nunca "*") y permite credenciales: el
// navegador manda la cookie de sesion de mapas con `credentials: "include"`.
func CORS(allowedOrigins []string) fiber.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		c.Vary(fiber.HeaderOrigin)

		origin := c.Get(fiber.HeaderOrigin)
		_, known := allowed[origin]
		if origin != "" && (allowAll || known) {
			c.Set(fiber.HeaderAccessControlAllowOrigin, origin)
			c.Set(fiber.HeaderAccessControlAllowCredentials, "true")
			c.Set(fiber.HeaderAccessControlAllowMethods, "GET,POST,OPTIONS")
			c.Set(fiber.HeaderAccessControlAllowHeaders, "Content-Type, Authorization")
			c.Set(fiber.HeaderAccessControlMaxAge, "600")
		}

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
