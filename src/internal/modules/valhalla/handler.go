package valhalla

import (
	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Infra *app.Infrastructure
}

func (h *Handler) Route(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return response.Error(c, "missing request body", fiber.StatusBadRequest)
	}

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL:    h.Infra.Config.Upstream.ValhallaURL,
		TargetPath:         "/route",
		Method:             "POST",
		Body:               body,
		ContentType:        "application/json",
		DefaultContentType: "application/json; charset=utf-8",
	})
}

func (h *Handler) TraceAttributes(c *fiber.Ctx) error {
	body := c.Body()
	if len(body) == 0 {
		return response.Error(c, "missing request body", fiber.StatusBadRequest)
	}

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL:    h.Infra.Config.Upstream.ValhallaURL,
		TargetPath:         "/trace_attributes",
		Method:             "POST",
		Body:               body,
		ContentType:        "application/json",
		DefaultContentType: "application/json; charset=utf-8",
	})
}
