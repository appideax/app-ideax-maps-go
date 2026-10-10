package routes

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/internal/modules/geocode"
	"app-ideax-maps-go/internal/modules/maps"
	"app-ideax-maps-go/internal/modules/valhalla"

	"github.com/gofiber/fiber/v2"
)

func Register(app *fiber.App, infra *app.Infrastructure) {
	registerHealth(app)

	maps.Register(app, infra)
	geocode.Register(app, infra)
	valhalla.Register(app, infra)
}
