package valhalla

import (
	"bytes"
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

func setupValhallaApp(t *testing.T, upstream *httptest.Server) (*fiber.App, string) {
	t.Helper()
	cfg := testutil.NewTestConfig()
	cfg.Upstream.ValhallaURL = upstream.URL
	infra := testutil.NewTestInfrastructure(cfg)

	token, err := middleware.BuildToken(cfg.JWT.Secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	Register(app, infra)
	return app, token
}

func TestHandler_Route(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/route", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"locations":[]}`, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"trip":{}}`))
	}))
	defer upstream.Close()

	app, token := setupValhallaApp(t, upstream)

	req := httptest.NewRequest(http.MethodPost, "/valhalla/route", bytes.NewReader([]byte(`{"locations":[]}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, `{"trip":{}}`, string(body))
	resp.Body.Close()
}

func TestHandler_Route_MissingBody(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	app, token := setupValhallaApp(t, upstream)

	req := httptest.NewRequest(http.MethodPost, "/valhalla/route", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	resp.Body.Close()
}
