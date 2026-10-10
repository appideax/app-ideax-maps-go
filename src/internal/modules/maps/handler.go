package maps

import (
	"fmt"
	"strings"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"
	"app-ideax-maps-go/shared/validator"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Infra *app.Infrastructure
}

func (h *Handler) Tiles(c *fiber.Ctx) error {
	z, err := validator.ParseZoom(c.Params("z"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}
	x, err := validator.ParseTileCoord(z, c.Params("x"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}
	y, err := validator.ParseTileCoord(z, c.Params("y"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	targetPath := fmt.Sprintf("/%d/%d/%d", z, x, y)
	cacheKey := fmt.Sprintf("tiles|%d|%d|%d", z, x, y)

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL:    h.Infra.Config.Upstream.TilesURL,
		TargetPath:         targetPath,
		CacheKey:           cacheKey,
		CacheTTL:           h.Infra.Config.Cache.TTLTiles,
		DefaultContentType: "application/x-protobuf",
	})
}

func (h *Handler) Fonts(c *fiber.Ctx) error {
	fontstack := c.Params("fontstack")
	rangeParam := c.Params("range")
	if fontstack == "" || rangeParam == "" || !strings.HasSuffix(rangeParam, ".pbf") {
		return response.Error(c, "invalid font request", fiber.StatusBadRequest)
	}

	targetPath := fmt.Sprintf("/%s/%s", fontstack, rangeParam)
	cacheKey := fmt.Sprintf("fonts|%s|%s", fontstack, rangeParam)

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL:    h.Infra.Config.Upstream.GlyphsURL,
		TargetPath:         targetPath,
		CacheKey:           cacheKey,
		CacheTTL:           h.Infra.Config.Cache.TTLGlyphs,
		DefaultContentType: "application/x-protobuf",
	})
}
