package geocode

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register monta la geocodificacion, que va toda por Pelias. /pelias/search es
// la misma busqueda que /geocode/search: se mantiene por los clientes que ya la usan.
func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	auth := middleware.Auth(infra.Config.Auth)
	limit := middleware.RateLimit("geocode", infra.Config.RateLimit.GeocodePerMinute)

	geocode := router.Group("/geocode", auth, limit)
	geocode.Get("/search", handler.Search)
	geocode.Get("/reverse", handler.Reverse)

	pelias := router.Group("/pelias", auth, limit)
	pelias.Get("/search", handler.Search)
}
