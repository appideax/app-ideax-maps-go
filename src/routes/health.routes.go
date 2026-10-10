package routes

import (
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func registerHealth(app *fiber.App) {
	app.Get("/healthz", middleware.HealthRateLimit(), func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})
}
