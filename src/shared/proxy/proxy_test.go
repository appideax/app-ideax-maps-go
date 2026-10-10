package proxy_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app-ideax-maps-go/internal/testutil"
	"app-ideax-maps-go/shared/proxy"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func forwardApp(opts proxy.Options) *fiber.App {
	infra := testutil.NewTestInfrastructure(nil)
	app := fiber.New()
	app.All("/test", func(c *fiber.Ctx) error {
		return proxy.Forward(c, infra, opts)
	})
	return app
}

func do(t *testing.T, app *fiber.App, req *http.Request) (*http.Response, string) {
	t.Helper()
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(body)
}

// El upstream manda las teselas en gzip: se guardan planas y un acierto de
// cache devuelve lo mismo que la primera respuesta.
func TestForward_CachesDecodedBody(t *testing.T) {
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		assert.Equal(t, "gzip", r.Header.Get("Accept-Encoding"))
		w.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(w)
		_, _ = zw.Write([]byte("tile-data"))
		_ = zw.Close()
	}))
	defer upstream.Close()

	app := forwardApp(proxy.Options{
		UpstreamBaseURL: upstream.URL,
		TargetPath:      "/1/1/1",
		ContentType:     proxy.ContentTypeProtobuf,
		CacheKey:        "tiles|1|1|1",
		CacheTTL:        time.Hour,
	})

	for range 2 {
		resp, body := do(t, app, httptest.NewRequest(http.MethodGet, "/test", nil))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "tile-data", body)
		assert.Equal(t, proxy.ContentTypeProtobuf, resp.Header.Get("Content-Type"))
		assert.Equal(t, "private, max-age=3600", resp.Header.Get("Cache-Control"))
		assert.Empty(t, resp.Header.Get("Content-Encoding"))
	}
	assert.Equal(t, 1, calls)
}

func TestForward_DoesNotLeakHeaders(t *testing.T) {
	var received http.Header
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Clone()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Set-Cookie", "upstream=1")
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	app := forwardApp(proxy.Options{UpstreamBaseURL: upstream.URL, TargetPath: "/x", ContentType: proxy.ContentTypeJSON})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("Cookie", "maps_session=abc")
	resp, _ := do(t, app, req)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Empty(t, received.Get("Authorization"))
	assert.Empty(t, received.Get("Cookie"))
	assert.Equal(t, proxy.UserAgent, received.Get("User-Agent"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Set-Cookie"))
	assert.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
}

func TestForward_UpstreamStatuses(t *testing.T) {
	cases := []struct {
		name     string
		upstream int
		want     int
	}{
		{"server error becomes bad gateway", http.StatusInternalServerError, http.StatusBadGateway},
		{"client error passes through", http.StatusBadRequest, http.StatusBadRequest},
		{"empty tile", http.StatusNoContent, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(tc.upstream)
				if tc.upstream != http.StatusNoContent {
					_, _ = w.Write([]byte(`{"error":"detail"}`))
				}
			}))
			defer upstream.Close()

			app := forwardApp(proxy.Options{
				UpstreamBaseURL: upstream.URL,
				TargetPath:      "/x",
				ContentType:     proxy.ContentTypeJSON,
				CacheKey:        "k",
				CacheTTL:        time.Hour,
			})

			for range 2 {
				resp, _ := do(t, app, httptest.NewRequest(http.MethodGet, "/test", nil))
				assert.Equal(t, tc.want, resp.StatusCode)
			}
			// Nada de esto se guarda en cache.
			assert.Equal(t, 2, calls)
		})
	}
}

func TestForward_UpstreamDown(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	upstream.Close()

	app := forwardApp(proxy.Options{UpstreamBaseURL: upstream.URL, TargetPath: "/x", ContentType: proxy.ContentTypeJSON})

	resp, _ := do(t, app, httptest.NewRequest(http.MethodGet, "/test", nil))
	assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
}
