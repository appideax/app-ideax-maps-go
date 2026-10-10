package valhalla

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	group := router.Group("/valhalla")
	group.Use(middleware.Auth(infra.Config.Auth))
	group.Use(middleware.RateLimit("routing", infra.Config.RateLimit.RoutingPerMinute))

	group.Post("/route", handler.Route)
	group.Post("/trace_attributes", handler.TraceAttributes)
}
