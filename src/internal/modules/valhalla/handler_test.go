package valhalla

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"app-ideax-maps-go/internal/testutil"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type upstreamCall struct {
	path, contentType, body string
}

func post(t *testing.T, path, body string) (int, string, *upstreamCall) {
	t.Helper()
	var call *upstreamCall
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		call = &upstreamCall{path: r.URL.Path, contentType: r.Header.Get("Content-Type"), body: string(raw)}
		_, _ = w.Write([]byte(`{"trip":{}}`))
	}))
	defer upstream.Close()

	cfg := testutil.NewTestConfig()
	cfg.Upstream.ValhallaURL = upstream.URL

	app := fiber.New()
	Register(app, testutil.NewTestInfrastructure(cfg))

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+testutil.Token(t, cfg))
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, string(raw), call
}

func TestRoute(t *testing.T) {
	status, body, call := post(t, "/valhalla/route", `{"locations":[]}`)

	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, `{"trip":{}}`, body)
	require.NotNil(t, call)
	assert.Equal(t, "/route", call.path)
	assert.Equal(t, "application/json", call.contentType)
	assert.Equal(t, `{"locations":[]}`, call.body)
}

func TestTraceRoute(t *testing.T) {
	status, _, call := post(t, "/valhalla/trace_route", `{"shape":[]}`)

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, call)
	assert.Equal(t, "/trace_route", call.path)
}

func TestTraceAttributes(t *testing.T) {
	status, _, call := post(t, "/valhalla/trace_attributes", `{"shape":[]}`)

	assert.Equal(t, http.StatusOK, status)
	require.NotNil(t, call)
	assert.Equal(t, "/trace_attributes", call.path)
}

func TestRoute_InvalidBody(t *testing.T) {
	for _, body := range []string{"", "not-json"} {
		status, _, call := post(t, "/valhalla/route", body)

		assert.Equal(t, http.StatusBadRequest, status)
		assert.Nil(t, call)
	}
}
