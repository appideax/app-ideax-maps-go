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

// Rectangulo de busqueda por defecto: Ecuador y una franja cercana.
var ecuadorBBox = validator.BBox{MinLon: -81.5, MinLat: -6.5, MaxLon: -74.5, MaxLat: 1.5}

type Handler struct {
	Infra *app.Infrastructure
}

// PhotonSearch: GET /geocode/search?q=&locale=&bbox=minLon,minLat,maxLon,maxLat
func (h *Handler) PhotonSearch(c *fiber.Ctx) error {
	q, ok := searchText(c)
	if !ok {
		return response.Error(c, "missing or invalid q parameter", fiber.StatusBadRequest)
	}

	// El cliente puede acotar la busqueda (p. ej. al barrio elegido); si el
	// rectangulo no sirve se busca en todo Ecuador.
	bbox, ok := validator.ParseBBox(c.Query("bbox"))
	if !ok {
		bbox = ecuadorBBox
	}

	query := url.Values{}
	query.Set("q", q)
	query.Set("lang", photonLang(c.Query("locale")))
	query.Set("limit", strconv.Itoa(searchLimit))
	query.Set("bbox", bbox.String())
	query.Set("countrycode", countryCode)

	return h.forward(c, h.Infra.Config.Upstream.PhotonURL, "/api", query, photonTransform)
}

// PhotonReverse: GET /geocode/reverse?lat=&lng=&locale=
func (h *Handler) PhotonReverse(c *fiber.Ctx) error {
	lat, lng, err := validator.ParseLatLng(c.Query("lat"), c.Query("lng"))
	if err != nil {
		return response.Error(c, err.Error(), fiber.StatusBadRequest)
	}

	// Cuatro decimales (unos 11 m): puntos cercanos comparten la entrada de cache.
	query := url.Values{}
	query.Set("lat", strconv.FormatFloat(lat, 'f', 4, 64))
	query.Set("lon", strconv.FormatFloat(lng, 'f', 4, 64))
	query.Set("lang", photonLang(c.Query("locale")))
	query.Set("limit", "1")

	return h.forward(c, h.Infra.Config.Upstream.PhotonURL, "/reverse", query, photonTransform)
}

// PeliasSearch: GET /pelias/search?q=&locale=&bbox=minLon,minLat,maxLon,maxLat
func (h *Handler) PeliasSearch(c *fiber.Ctx) error {
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

	return h.forward(c, h.Infra.Config.Upstream.PeliasURL, "/v1/search", query, peliasTransform)
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

func locale(raw string) string {
	switch raw = strings.ToLower(strings.TrimSpace(raw)); raw {
	case "es", "en":
		return raw
	default:
		return defaultLocale
	}
}

// Photon solo responde en estos idiomas; cualquier otro va como "default".
func photonLang(raw string) string {
	switch raw = strings.ToLower(strings.TrimSpace(raw)); raw {
	case "de", "en", "fr":
		return raw
	default:
		return "default"
	}
}
