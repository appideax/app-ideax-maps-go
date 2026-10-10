package pelias

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	group := router.Group("/pelias")
	group.Use(middleware.Auth(infra.Config.JWT, infra.Config.Map.CookieName))
	group.Use(middleware.PeliasRateLimit(infra.Config.RateLimit.PerMinute))

	group.Get("/search", handler.Search)
	group.Get("/autocomplete", handler.Autocomplete)
	group.Get("/reverse", handler.Reverse)
	group.Get("/structured", handler.Structured)
	group.Get("/place", handler.Place)
}
