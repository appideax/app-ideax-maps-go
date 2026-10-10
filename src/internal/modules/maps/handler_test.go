package maps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"app-ideax-maps-go/internal/testutil"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// request llama a path con un token valido; upstreamPath recibe la ruta que
// llego a Martin ("" si no se llamo).
func request(t *testing.T, path string, authenticated bool) (status int, body, upstreamPath string) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte("data"))
	}))
	defer upstream.Close()

	cfg := testutil.NewTestConfig()
	cfg.Upstream.TilesURL = upstream.URL + "/ecuador"
	cfg.Upstream.GlyphsURL = upstream.URL + "/font"

	app := fiber.New()
	Register(app, testutil.NewTestInfrastructure(cfg))

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authenticated {
		req.Header.Set("Authorization", "Bearer "+testutil.Token(t, cfg))
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw), upstreamPath
}

func TestTiles(t *testing.T) {
	status, body, upstreamPath := request(t, "/maps/tiles/2/1/1", true)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "data", body)
	assert.Equal(t, "/ecuador/2/1/1", upstreamPath)
}

func TestTiles_RequiresToken(t *testing.T) {
	status, _, upstreamPath := request(t, "/maps/tiles/2/1/1", false)

	assert.Equal(t, http.StatusUnauthorized, status)
	assert.Empty(t, upstreamPath)
}

func TestTiles_InvalidCoord(t *testing.T) {
	for _, path := range []string{"/maps/tiles/2/9/1", "/maps/tiles/99/1/1", "/maps/tiles/2/a/1"} {
		status, _, upstreamPath := request(t, path, true)

		assert.Equal(t, http.StatusBadRequest, status, path)
		assert.Empty(t, upstreamPath, path)
	}
}

// MapLibre pide ".pbf" y el nombre de la fuente codificado; a Martin se le
// pide sin extension y con el nombre codificado una sola vez.
func TestFonts(t *testing.T) {
	status, body, upstreamPath := request(t, "/maps/fonts/Noto%20Sans%20Regular/0-255.pbf", true)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "data", body)
	assert.Equal(t, "/font/Noto%20Sans%20Regular/0-255", upstreamPath)
}

func TestFonts_Invalid(t *testing.T) {
	for _, path := range []string{
		"/maps/fonts/Noto%20Sans%20Regular/0-255",
		"/maps/fonts/Noto%20Sans%20Regular/abc.pbf",
		"/maps/fonts/..%2F..%2Fsecret/0-255.pbf",
		"/maps/fonts/a%3Fb/0-255.pbf",
	} {
		status, _, upstreamPath := request(t, path, true)

		assert.Equal(t, http.StatusBadRequest, status, path)
		assert.Empty(t, upstreamPath, path)
	}
}
