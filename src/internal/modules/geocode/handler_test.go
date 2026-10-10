package geocode

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"app-ideax-maps-go/internal/testutil"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type upstreamCall struct {
	path  string
	query url.Values
}

// get llama a path con un token valido contra un upstream que responde payload.
func get(t *testing.T, path, payload string) (int, string, *upstreamCall) {
	t.Helper()
	var call *upstreamCall
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call = &upstreamCall{path: r.URL.Path, query: r.URL.Query()}
		_, _ = w.Write([]byte(payload))
	}))
	defer upstream.Close()

	cfg := testutil.NewTestConfig()
	cfg.Upstream.PhotonURL = upstream.URL
	cfg.Upstream.PeliasURL = upstream.URL

	app := fiber.New()
	Register(app, testutil.NewTestInfrastructure(cfg))

	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+testutil.Token(t, cfg))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw), call
}

func TestPhotonSearch(t *testing.T) {
	status, body, call := get(t, "/geocode/search?q=av+amazonas&locale=es", `{"features":[]}`)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, `{"results":[]}`, body)
	require.NotNil(t, call)
	assert.Equal(t, "/api", call.path)
	assert.Equal(t, "av amazonas", call.query.Get("q"))
	assert.Equal(t, "default", call.query.Get("lang"))
	assert.Equal(t, "8", call.query.Get("limit"))
	assert.Equal(t, "EC", call.query.Get("countrycode"))
	assert.Equal(t, "-81.5,-6.5,-74.5,1.5", call.query.Get("bbox"))
}

func TestPhotonSearch_ClientBBoxAndLang(t *testing.T) {
	_, _, call := get(t, "/geocode/search?q=quito&locale=en&bbox=-78.6,-0.4,-78.3,0", `{"features":[]}`)

	require.NotNil(t, call)
	assert.Equal(t, "en", call.query.Get("lang"))
	assert.Equal(t, "-78.6,-0.4,-78.3,0", call.query.Get("bbox"))
}

func TestPhotonSearch_InvalidBBoxFallsBackToEcuador(t *testing.T) {
	_, _, call := get(t, "/geocode/search?q=quito&bbox=1,2,3", `{"features":[]}`)

	require.NotNil(t, call)
	assert.Equal(t, "-81.5,-6.5,-74.5,1.5", call.query.Get("bbox"))
}

func TestSearch_MissingQuery(t *testing.T) {
	for _, path := range []string{"/geocode/search", "/geocode/search?q=+", "/pelias/search"} {
		status, _, call := get(t, path, `{"features":[]}`)

		assert.Equal(t, http.StatusBadRequest, status, path)
		assert.Nil(t, call, path)
	}
}

func TestPhotonReverse(t *testing.T) {
	status, body, call := get(t, "/geocode/reverse?lat=-0.18065312&lng=-78.46783812&locale=es",
		`{"features":[{"geometry":{"coordinates":[-78.4678,-0.1807]},"properties":{"osm_type":"W","osm_id":123,"name":"Parque La Carolina","street":"Av. Amazonas","housenumber":"N34","city":"Quito","state":"Pichincha","country":"Ecuador"}}]}`)

	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"results":[{"id":"W123","address":"Parque La Carolina, Av. Amazonas N34, Quito, Pichincha","latitude":-0.1807,"longitude":-78.4678}]}`, body)
	require.NotNil(t, call)
	assert.Equal(t, "/reverse", call.path)
	assert.Equal(t, "-0.1807", call.query.Get("lat"))
	assert.Equal(t, "-78.4678", call.query.Get("lon"))
	assert.Equal(t, "1", call.query.Get("limit"))
}

func TestPhotonReverse_InvalidCoords(t *testing.T) {
	for _, path := range []string{"/geocode/reverse", "/geocode/reverse?lat=91&lng=0", "/geocode/reverse?lat=0&lng=abc", "/geocode/reverse?lat=NaN&lng=0"} {
		status, _, call := get(t, path, `{"features":[]}`)

		assert.Equal(t, http.StatusBadRequest, status, path)
		assert.Nil(t, call, path)
	}
}

func TestPeliasSearch(t *testing.T) {
	status, body, call := get(t, "/pelias/search?q=amazonas+y+naciones+unidas&locale=es&bbox=-78.6,-0.4,-78.3,0",
		`{"features":[
			{"geometry":{"coordinates":[-78.48,-0.17]},"properties":{"gid":"osm:1","name":"Av. Amazonas","locality":"Quito","region":"Pichincha"}},
			{"geometry":{"coordinates":[-78.48,-0.17]},"properties":{"gid":"osm:1","name":"Repetido"}},
			{"geometry":{"coordinates":[-78.5,-0.2]},"properties":{}},
			{"geometry":{},"properties":{"gid":"osm:2","name":"Sin coordenadas"}}
		]}`)

	assert.Equal(t, http.StatusOK, status)
	assert.JSONEq(t, `{"results":[
		{"id":"pelias:osm:1","address":"Av. Amazonas, Quito, Pichincha","latitude":-0.17,"longitude":-78.48},
		{"id":"-0.20000, -78.50000","address":"-0.20000, -78.50000","latitude":-0.2,"longitude":-78.5}
	]}`, body)
	require.NotNil(t, call)
	assert.Equal(t, "/v1/search", call.path)
	assert.Equal(t, "amazonas y naciones unidas", call.query.Get("text"))
	assert.Equal(t, "8", call.query.Get("size"))
	assert.Equal(t, "es", call.query.Get("lang"))
	assert.Equal(t, "EC", call.query.Get("boundary.country"))
	assert.Equal(t, "-78.6", call.query.Get("boundary.rect.min_lon"))
	assert.Equal(t, "0", call.query.Get("boundary.rect.max_lat"))
}

func TestUpstreamGarbageIsBadGateway(t *testing.T) {
	status, _, _ := get(t, "/geocode/search?q=quito", `<html>`)

	assert.Equal(t, http.StatusBadGateway, status)
}
