package geocode

import (
	"net/url"
	"strconv"
	"strings"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/proxy"
	"app-ideax-maps-go/shared/response"
	"app-ideax-maps-go/shared/validator"

	"github.com/gofiber/fiber/v2"
)

const (
	searchLimit    = 8
	maxQueryLength = 200
	countryCode    = "EC"
	defaultLocale  = "es"
)

type Handler struct {
	Infra *app.Infrastructure
}

// Reverse: GET /geocode/reverse?lat=&lng=&locale=
// Sin capas: Pelias devuelve la direccion o el lugar mas cercano y, si no hay
// ninguno cerca, cae solo al barrio o la ciudad.
func (h *Handler) Reverse(c *fiber.Ctx) error {
	lat, lng, err := validator.ParseLatLng(c.Query("lat"), c.Query("lng"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	// Cuatro decimales (unos 11 m): puntos cercanos comparten la entrada de cache.
	query := url.Values{}
	query.Set("point.lat", strconv.FormatFloat(lat, 'f', 4, 64))
	query.Set("point.lon", strconv.FormatFloat(lng, 'f', 4, 64))
	query.Set("lang", locale(c.Query("locale")))
	query.Set("size", "1")

	return h.forward(c, h.Infra.Config.Upstream.PeliasURL, "/v1/reverse", query, peliasTransform)
}

// Search: GET /geocode/search?q=&locale=&bbox=minLon,minLat,maxLon,maxLat&focus=lat,lng
// Va por autocomplete: empata por prefijo mientras se escribe y encuentra los
// lugares por nombre (edificios, centros comerciales), no solo direcciones.
func (h *Handler) Search(c *fiber.Ctx) error {
	q, ok := searchText(c)
	if !ok {
		return response.Error(c, "missing or invalid q parameter", fiber.StatusBadRequest)
	}

	query := url.Values{}
	query.Set("text", q)
	query.Set("size", strconv.Itoa(searchLimit))
	query.Set("lang", locale(c.Query("locale")))
	query.Set("boundary.country", countryCode)
	if bbox, ok := validator.ParseBBox(c.Query("bbox")); ok {
		query.Set("boundary.rect.min_lon", strconv.FormatFloat(bbox.MinLon, 'g', -1, 64))
		query.Set("boundary.rect.min_lat", strconv.FormatFloat(bbox.MinLat, 'g', -1, 64))
		query.Set("boundary.rect.max_lon", strconv.FormatFloat(bbox.MaxLon, 'g', -1, 64))
		query.Set("boundary.rect.max_lat", strconv.FormatFloat(bbox.MaxLat, 'g', -1, 64))
	}
	if lat, lng, ok := focusPoint(c); ok {
		query.Set("focus.point.lat", lat)
		query.Set("focus.point.lon", lng)
	}

	return h.search(c, h.Infra.Config.Upstream.PeliasURL, "/v1/autocomplete", query, "text", peliasTransform)
}

// search es forward con reintentos: si el texto no trae resultados se prueban
// sus variantes (ver alternatives). textKey es el parametro que lleva el texto.
func (h *Handler) search(c *fiber.Ctx, baseURL, targetPath string, query url.Values, textKey string, transform func([]byte) ([]byte, error)) error {
	var fallbacks []url.Values
	for _, text := range alternatives(query.Get(textKey)) {
		alt := url.Values{}
		for key, values := range query {
			alt[key] = values
		}
		alt.Set(textKey, text)
		fallbacks = append(fallbacks, alt)
	}

	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL: baseURL,
		TargetPath:      targetPath,
		Query:           query,
		ContentType:     proxy.ContentTypeJSON,
		CacheKey:        baseURL + targetPath + "?" + query.Encode(),
		CacheTTL:        h.Infra.Config.Cache.TTLGeocode,
		Transform:       transform,
		Fallbacks:       fallbacks,
		IsEmpty:         isEmpty,
	})
}

func (h *Handler) forward(c *fiber.Ctx, baseURL, targetPath string, query url.Values, transform func([]byte) ([]byte, error)) error {
	return proxy.Forward(c, h.Infra, proxy.Options{
		UpstreamBaseURL: baseURL,
		TargetPath:      targetPath,
		Query:           query,
		ContentType:     proxy.ContentTypeJSON,
		CacheKey:        baseURL + targetPath + "?" + query.Encode(),
		CacheTTL:        h.Infra.Config.Cache.TTLGeocode,
		Transform:       transform,
	})
}

func searchText(c *fiber.Ctx) (string, bool) {
	q := strings.TrimSpace(c.Query("q"))
	return q, q != "" && len(q) <= maxQueryLength
}

// focusPoint lee focus=lat,lng: el punto alrededor del cual se priorizan los
// resultados, sin descartar los lejanos. Dos decimales (alrededor de 1 km):
// quienes buscan desde la misma zona comparten la entrada de cache.
func focusPoint(c *fiber.Ctx) (lat, lng string, ok bool) {
	rawLat, rawLng, found := strings.Cut(c.Query("focus"), ",")
	if !found {
		return "", "", false
	}
	latitude, longitude, err := validator.ParseLatLng(strings.TrimSpace(rawLat), strings.TrimSpace(rawLng))
	if err != nil {
		return "", "", false
	}
	return strconv.FormatFloat(latitude, 'f', 2, 64), strconv.FormatFloat(longitude, 'f', 2, 64), true
}

func locale(raw string) string {
	switch raw = strings.ToLower(strings.TrimSpace(raw)); raw {
	case "es", "en":
		return raw
	default:
		return defaultLocale
	}
}
