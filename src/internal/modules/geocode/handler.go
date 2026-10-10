package geocode

import (
	"fmt"
	"net/url"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/geocode"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"
	"app-ideax-maps-go/shared/validator"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Infra *app.Infrastructure
}

func (h *Handler) Search(c *fiber.Ctx) error {
	params := proxy.QueryParams(c)
	q := params.Get("q")
	if q == "" {
		return response.Error(c, "missing q parameter", fiber.StatusBadRequest)
	}

	limit, err := validator.ParseLimit(params.Get("limit"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("q", q)
	query.Set("limit", fmt.Sprintf("%d", limit))
	if lang := params.Get("lang"); lang != "" {
		query.Set("lang", lang)
	}

	return forward(c, h.Infra, "/api", query)
}

func (h *Handler) Reverse(c *fiber.Ctx) error {
	params := proxy.QueryParams(c)
	lat, lon, err := validator.ParseLatLon(params.Get("lat"), params.Get("lon"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	limit, err := validator.ParseLimit(params.Get("limit"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("lat", fmt.Sprintf("%f", lat))
	query.Set("lon", fmt.Sprintf("%f", lon))
	query.Set("limit", fmt.Sprintf("%d", limit))
	if lang := params.Get("lang"); lang != "" {
		query.Set("lang", lang)
	}

	return forward(c, h.Infra, "/reverse", query)
}

func (h *Handler) Structured(c *fiber.Ctx) error {
	allowed := []string{"street", "city", "county", "state", "country", "postalcode", "limit", "lang"}
	query := proxy.FilterQuery(proxy.QueryParams(c), allowed)

	return forward(c, h.Infra, "/structured", query)
}

func forward(c *fiber.Ctx, infra *app.Infrastructure, targetPath string, query url.Values) error {
	cacheKey := proxy.CacheKey("geocode", targetPath, query.Encode())
	return proxy.Forward(c, infra, proxy.Options{
		UpstreamBaseURL:    infra.Config.Upstream.PhotonURL,
		TargetPath:         targetPath,
		QueryParams:        query,
		CacheKey:           cacheKey,
		CacheTTL:           infra.Config.Cache.TTLGeocode,
		DefaultContentType: "application/json; charset=utf-8",
		Transform:          geocode.PhotonTransform("photon"),
	})
}
