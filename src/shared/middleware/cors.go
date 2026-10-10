package middleware

import (
	"github.com/gofiber/fiber/v2"
)

func CORS(allowedOrigins []string) fiber.Handler {
	allowAll := false
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowed[origin] = struct{}{}
	}

	allowMethods := "GET,POST,OPTIONS"
	allowHeaders := "Content-Type, Authorization"

	return func(c *fiber.Ctx) error {
		allowOrigin := ""
		origin := c.Get("Origin")
		if allowAll {
			allowOrigin = "*"
		} else if origin != "" {
			if _, ok := allowed[origin]; ok {
				allowOrigin = origin
			}
		}

		c.Set("Vary", "Origin")
		if allowOrigin != "" {
			c.Set("Access-Control-Allow-Origin", allowOrigin)
			c.Set("Access-Control-Allow-Methods", allowMethods)
			c.Set("Access-Control-Allow-Headers", allowHeaders)
		}

		if c.Method() == fiber.MethodOptions {
			c.Status(fiber.StatusNoContent)
			return nil
		}

		return c.Next()
	}
}
