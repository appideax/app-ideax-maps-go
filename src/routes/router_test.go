package routes

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

func TestRegister_Healthz(t *testing.T) {
	infra := testutil.NewTestInfrastructure(nil)
	app := fiber.New()
	Register(app, infra)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"ok"`)
	resp.Body.Close()
}
