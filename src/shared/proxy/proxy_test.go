package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/cache"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInfrastructure() *app.Infrastructure {
	return &app.Infrastructure{
		Config: &config.Config{},
		Cache:  cache.NewMemoryCache(time.Hour),
	}
}

func TestForward_CacheHit(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("cached-data"))
	}))
	defer upstream.Close()

	infra := testInfrastructure()

	app := fiber.New()
	app.Get("/tiles/:z/:x/:y", func(c *fiber.Ctx) error {
		return Forward(c, infra, Options{
			UpstreamBaseURL:    upstream.URL,
			TargetPath:         "/1/1/1",
			CacheKey:           "tiles|1|1|1",
			CacheTTL:           time.Hour,
			DefaultContentType: "application/x-protobuf",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/tiles/1/1/1", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "cached-data", string(body))
	resp.Body.Close()

	req = httptest.NewRequest(http.MethodGet, "/tiles/1/1/1", nil)
	resp, err = app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ = io.ReadAll(resp.Body)
	assert.Equal(t, "cached-data", string(body))
	resp.Body.Close()

	assert.Equal(t, 1, calls)
}

func TestForward_StripsSensitiveHeaders(t *testing.T) {
	var received http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	infra := testInfrastructure()

	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return Forward(c, infra, Options{
			UpstreamBaseURL: upstream.URL,
			TargetPath:      "/test",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "session=abc")
	req.Header.Set("Accept-Encoding", "gzip")
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	assert.Empty(t, received.Get("Authorization"))
	assert.Empty(t, received.Get("Cookie"))
	assert.Equal(t, "AppIdeaxMapsProxy/1.0", received.Get("User-Agent"))
	assert.Equal(t, "gzip", received.Get("Accept-Encoding"))
}

func TestFilterQuery(t *testing.T) {
	params := url.Values{}
	params.Set("q", "quito")
	params.Set("foo", "bar")

	filtered := FilterQuery(params, []string{"q"})
	assert.Equal(t, "quito", filtered.Get("q"))
	assert.Empty(t, filtered.Get("foo"))
}

func TestCacheKey(t *testing.T) {
	key := CacheKey("a", "b", "c")
	assert.Equal(t, "a|b|c", key)
}
