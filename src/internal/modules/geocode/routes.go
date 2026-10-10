package geocode

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	group := router.Group("/geocode")
	group.Use(middleware.Auth(infra.Config.JWT, infra.Config.Map.CookieName))
	group.Use(middleware.GeocodeRateLimit(infra.Config.RateLimit.PerMinute))

	group.Get("/search", handler.Search)
	group.Get("/reverse", handler.Reverse)
	group.Get("/structured", handler.Structured)
}
