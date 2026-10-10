package maps

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	group := router.Group("/maps")
	group.Use(middleware.Auth(infra.Config.JWT, infra.Config.Map.CookieName))
	group.Use(middleware.MapRateLimit(infra.Config.RateLimit.PerMinute))

	group.Get("/tiles/:z/:x/:y", handler.Tiles)
	group.Get("/fonts/:fontstack/:range", handler.Fonts)
}
