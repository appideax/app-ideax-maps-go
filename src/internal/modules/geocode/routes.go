package geocode

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
)

// Register monta la geocodificacion. Los dos buscadores comparten contrato
// (mismos parametros, misma respuesta); el reverse va siempre por Photon.
func Register(router fiber.Router, infra *app.Infrastructure) {
	handler := &Handler{Infra: infra}

	auth := middleware.Auth(infra.Config.Auth)
	limit := middleware.RateLimit("geocode", infra.Config.RateLimit.GeocodePerMinute)

	photon := router.Group("/geocode", auth, limit)
	photon.Get("/search", handler.PhotonSearch)
	photon.Get("/reverse", handler.PhotonReverse)

	pelias := router.Group("/pelias", auth, limit)
	pelias.Get("/search", handler.PeliasSearch)
}
