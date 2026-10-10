package pelias

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

func setupPeliasApp(t *testing.T, upstream *httptest.Server) (*fiber.App, string) {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.Upstream.PeliasURL = upstream.URL
	infra := testutil.NewTestInfrastructure(cfg)

	token, err := middleware.BuildToken(cfg.JWT.Secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	Register(app, infra)
	return app, token
}

func TestHandler_Search(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/search", r.URL.Path)
		assert.Equal(t, "quito", r.URL.Query().Get("text"))
		assert.Equal(t, "10", r.URL.Query().Get("size"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer upstream.Close()

	app, token := setupPeliasApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/pelias/search?text=quito", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"results":[]}`, string(body))
	resp.Body.Close()
}

func TestHandler_Place(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/place", r.URL.Path)
		assert.Equal(t, "osm:node:123", r.URL.Query().Get("ids"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"features":[]}`))
	}))
	defer upstream.Close()

	app, token := setupPeliasApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/pelias/place?ids=osm:node:123", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_Place_MissingIDs(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	app, token := setupPeliasApp(t, upstream)

	req := httptest.NewRequest(http.MethodGet, "/pelias/place", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}
