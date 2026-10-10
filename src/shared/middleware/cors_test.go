package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func corsResponse(t *testing.T, allowed []string, method, origin string) *http.Response {
	t.Helper()
	app := fiber.New()
	app.Use(CORS(allowed))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("ok") })

	req := httptest.NewRequest(method, "/test", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	resp.Body.Close()
	return resp
}

func TestCORS_AllowedOriginGetsCredentials(t *testing.T) {
	resp := corsResponse(t, []string{"https://dashboard.appideax.com"}, http.MethodGet, "https://dashboard.appideax.com")

	assert.Equal(t, "https://dashboard.appideax.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", resp.Header.Get("Vary"))
}

func TestCORS_UnknownOriginGetsNothing(t *testing.T) {
	resp := corsResponse(t, []string{"https://dashboard.appideax.com"}, http.MethodGet, "https://evil.example.com")

	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, resp.Header.Get("Access-Control-Allow-Credentials"))
}

// Con credenciales el navegador rechaza "*": se devuelve el origen que llama.
func TestCORS_WildcardEchoesOrigin(t *testing.T) {
	resp := corsResponse(t, []string{"*"}, http.MethodGet, "http://localhost:3000")

	assert.Equal(t, "http://localhost:3000", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

func TestCORS_Preflight(t *testing.T) {
	resp := corsResponse(t, []string{"https://appideax.com"}, http.MethodOptions, "https://appideax.com")

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "https://appideax.com", resp.Header.Get("Access-Control-Allow-Origin"))
	assert.Contains(t, resp.Header.Get("Access-Control-Allow-Headers"), "Authorization")
}
