package valhalla

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	group := router.Group("/valhalla")
	group.Use(middleware.Auth(infra.Config.JWT, infra.Config.Map.CookieName))
	group.Use(middleware.ValhallaRateLimit(infra.Config.RateLimit.PerMinute))

	group.Post("/route", handler.Route)
	group.Post("/trace_attributes", handler.TraceAttributes)
}
