package geocode

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

func setupGeocodeApp(t *testing.T, upstream *httptest.Server) (*fiber.App, string) {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.Upstream.PhotonURL = upstream.URL
	infra := testutil.NewTestInfrastructure(cfg)

	token, err := middleware.BuildToken(cfg.JWT.Secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	Register(app, infra)
	return app, token
}

func TestHandler_Search(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api", r.URL.Path)
		assert.Equal(t, "quito", r.URL.Query().Get("q"))
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer upstream.Close()

	app, token := setupGeocodeApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/geocode/search?q=quito", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"results":[]}`, string(body))
	resp.Body.Close()
}

func TestHandler_Reverse(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/reverse", r.URL.Path)
		assert.Equal(t, "-0.180700", r.URL.Query().Get("lat"))
		assert.Equal(t, "-78.467800", r.URL.Query().Get("lon"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer upstream.Close()

	app, token := setupGeocodeApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/geocode/reverse?lat=-0.1807&lon=-78.4678", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_Reverse_InvalidLatLon(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	app, token := setupGeocodeApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/geocode/reverse?lat=91&lon=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}
