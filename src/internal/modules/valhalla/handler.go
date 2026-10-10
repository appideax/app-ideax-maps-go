package valhalla

import (
	"encoding/json"
	"net/http"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Infra *app.Infrastructure
}

// Route: POST /valhalla/route
func (h *Handler) Route(c *fiber.Ctx) error {
	return h.forward(c, "/route")
}

// TraceRoute: POST /valhalla/trace_route. Lo usa taxi-back-go para ajustar a
// las calles el recorrido de una carrera.
func (h *Handler) TraceRoute(c *fiber.Ctx) error {
	return h.forward(c, "/trace_route")
}

// TraceAttributes: POST /valhalla/trace_attributes
func (h *Handler) TraceAttributes(c *fiber.Ctx) error {
	return h.forward(c, "/trace_attributes")
}

// forward reenvia el pedido JSON a Valhalla tal cual. No se cachea: cada ruta
// sale de la posicion del momento.
func (h *Handler) forward(c *fiber.Ctx, targetPath string) error {
	body := c.Body()
	if !json.Valid(body) {
		return response.Error(c, "request body must be valid JSON", fiber.StatusBadRequest)
	}

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL: h.Infra.Config.Upstream.ValhallaURL,
		TargetPath:      targetPath,
		Method:          http.MethodPost,
		Body:            body,
		ContentType:     proxy.ContentTypeJSON,
	})
}
