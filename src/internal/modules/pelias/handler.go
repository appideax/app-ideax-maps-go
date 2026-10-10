package pelias

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
	q := params.Get("text")
	if q == "" {
		return response.Error(c, "missing text parameter", fiber.StatusBadRequest)
	}

	limit, err := validator.ParseLimit(params.Get("size"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("text", q)
	query.Set("size", fmt.Sprintf("%d", limit))
	for _, key := range []string{"lang", "boundary.country", "sources", "layers"} {
		if v := params.Get(key); v != "" {
			query.Set(key, v)
		}
	}

	return forward(c, h.Infra, "/v1/search", query)
}

func (h *Handler) Autocomplete(c *fiber.Ctx) error {
	params := proxy.QueryParams(c)
	q := params.Get("text")
	if q == "" {
		return response.Error(c, "missing text parameter", fiber.StatusBadRequest)
	}

	limit, err := validator.ParseLimit(params.Get("size"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("text", q)
	query.Set("size", fmt.Sprintf("%d", limit))
	for _, key := range []string{"lang", "boundary.country", "sources", "layers", "focus.point.lat", "focus.point.lon"} {
		if v := params.Get(key); v != "" {
			query.Set(key, v)
		}
	}

	return forward(c, h.Infra, "/v1/autocomplete", query)
}

func (h *Handler) Reverse(c *fiber.Ctx) error {
	params := proxy.QueryParams(c)
	lat, lon, err := validator.ParseLatLon(params.Get("point.lat"), params.Get("point.lon"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	limit, err := validator.ParseLimit(params.Get("size"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("point.lat", fmt.Sprintf("%f", lat))
	query.Set("point.lon", fmt.Sprintf("%f", lon))
	query.Set("size", fmt.Sprintf("%d", limit))
	for _, key := range []string{"lang", "boundary.country", "sources", "layers"} {
		if v := params.Get(key); v != "" {
			query.Set(key, v)
		}
	}

	return forward(c, h.Infra, "/v1/reverse", query)
}

func (h *Handler) Structured(c *fiber.Ctx) error {
	allowed := []string{"address", "neighbourhood", "borough", "locality", "county", "region", "postalcode", "country", "size", "lang"}
	query := proxy.FilterQuery(proxy.QueryParams(c), allowed)

	return forward(c, h.Infra, "/v1/search/structured", query)
}

func (h *Handler) Place(c *fiber.Ctx) error {
	params := proxy.QueryParams(c)
	ids := params.Get("ids")
	if ids == "" {
		return response.Error(c, "missing ids parameter", fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("ids", ids)
	if lang := params.Get("lang"); lang != "" {
		query.Set("lang", lang)
	}

	return forward(c, h.Infra, "/v1/place", query)
}

func forward(c *fiber.Ctx, infra *app.Infrastructure, targetPath string, query url.Values) error {
	cacheKey := proxy.CacheKey("pelias", targetPath, query.Encode())
	return proxy.Forward(c, infra, proxy.Options{
		UpstreamBaseURL:    infra.Config.Upstream.PeliasURL,
		TargetPath:         targetPath,
		QueryParams:        query,
		CacheKey:           cacheKey,
		CacheTTL:           infra.Config.Cache.TTLGeocode,
		DefaultContentType: "application/json; charset=utf-8",
		Transform:          geocode.PeliasTransform("pelias"),
	})
}
