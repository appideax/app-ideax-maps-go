package maps

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app-ideax-maps-go/internal/testutil"
	"app-ideax-maps-go/shared/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMapsApp(t *testing.T, upstream *httptest.Server) (*fiber.App, string) {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.Upstream.TilesURL = upstream.URL + "/ecuador"
	cfg.Upstream.GlyphsURL = upstream.URL + "/font"
	infra := testutil.NewTestInfrastructure(cfg)

	token, err := middleware.BuildToken(cfg.JWT.Secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	Register(app, infra)
	return app, token
}

func TestHandler_Tiles(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ecuador/2/1/1", r.URL.Path)
		assert.Equal(t, "AppIdeaxMapsProxy/1.0", r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("tile-data"))
	}))
	defer upstream.Close()

	app, token := setupMapsApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/maps/tiles/2/1/1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "tile-data", string(body))
	resp.Body.Close()
}

func TestHandler_Tiles_InvalidCoord(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	app, token := setupMapsApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/maps/tiles/2/1/999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_Fonts(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/font/OpenSansRegular/0-255.pbf", r.URL.Path)
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("font-data"))
	}))
	defer upstream.Close()

	app, token := setupMapsApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/maps/fonts/OpenSansRegular/0-255.pbf", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, "font-data", string(body))
	resp.Body.Close()
}
