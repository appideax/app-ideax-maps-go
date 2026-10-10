package maps

import (
	"fmt"
	"net/url"
	"regexp"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"
	"app-ideax-maps-go/shared/validator"

	"github.com/gofiber/fiber/v2"
)

var (
	// Una o varias fuentes separadas por coma, p. ej. "Noto Sans Regular".
	fontstackPattern = regexp.MustCompile(`^[A-Za-z0-9 _-]+(,[A-Za-z0-9 _-]+)*$`)
	// Rango de glifos como lo pide MapLibre: "0-255.pbf".
	glyphRangePattern = regexp.MustCompile(`^(\d{1,5}-\d{1,5})\.pbf$`)
)

type Handler struct {
	Infra *app.Infrastructure
}

// Tiles: GET /maps/tiles/:z/:x/:y
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

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL: h.Infra.Config.Upstream.TilesURL,
		TargetPath:      fmt.Sprintf("/%d/%d/%d", z, x, y),
		ContentType:     proxy.ContentTypeProtobuf,
		CacheKey:        fmt.Sprintf("tiles|%d|%d|%d", z, x, y),
		CacheTTL:        h.Infra.Config.Cache.TTLTiles,
	})
}

// Fonts: GET /maps/fonts/:fontstack/:range.pbf
func (h *Handler) Fonts(c *fiber.Ctx) error {
	// El fontstack llega codificado en la ruta ("Noto%20Sans%20Regular").
	fontstack, err := url.PathUnescape(c.Params("fontstack"))
	glyphRange := glyphRangePattern.FindStringSubmatch(c.Params("range"))
	if err != nil || !fontstackPattern.MatchString(fontstack) || glyphRange == nil {
		return response.Error(c, "invalid font request", fiber.StatusBadRequest)
	}

	// Martin sirve los glifos sin la extension: con ".pbf" responde un redirect.
	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL: h.Infra.Config.Upstream.GlyphsURL,
		TargetPath:      "/" + url.PathEscape(fontstack) + "/" + glyphRange[1],
		ContentType:     proxy.ContentTypeProtobuf,
		CacheKey:        "fonts|" + fontstack + "|" + glyphRange[1],
		CacheTTL:        h.Infra.Config.Cache.TTLGlyphs,
	})
}
