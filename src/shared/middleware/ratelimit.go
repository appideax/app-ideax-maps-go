package middleware

import (
	"time"

	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

func sessionKey(c *fiber.Ctx) string {
	if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
		return uid
	}
	if rid, ok := c.Locals("ride_id").(string); ok && rid != "" {
		return rid
	}
	return ClientIP(c)
}

func HealthRateLimit(maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:health:" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusTooManyRequests)
		},
	})
}

func MapRateLimit(maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:maps:" + sessionKey(c) + ":" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, "Too many requests, please try again later", fiber.StatusTooManyRequests)
		},
	})
}

func GeocodeRateLimit(maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:geocode:" + sessionKey(c) + ":" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, "Too many requests, please try again later", fiber.StatusTooManyRequests)
		},
	})
}

func PeliasRateLimit(maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:pelias:" + sessionKey(c) + ":" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, "Too many requests, please try again later", fiber.StatusTooManyRequests)
		},
	})
}

func ValhallaRateLimit(maxPerMinute int) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        maxPerMinute,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return "ratelimit:valhalla:" + sessionKey(c) + ":" + ClientIP(c)
		},
		LimitReached: func(c *fiber.Ctx) error {
			return response.Error(c, "Too many requests, please try again later", fiber.StatusTooManyRequests)
		},
	})
}
