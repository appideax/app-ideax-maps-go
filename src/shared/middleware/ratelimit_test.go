package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// statuses hace n pedidos seguidos como subject, con un limite de 2 por minuto.
func statuses(t *testing.T, subject string, n int) []int {
	t.Helper()
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals(SubjectLocalKey, subject)
		return c.Next()
	})
	app.Use(RateLimit("test", 2))
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendString("ok") })

	codes := make([]int, 0, n)
	for range n {
		resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/test", nil), -1)
		require.NoError(t, err)
		resp.Body.Close()
		codes = append(codes, resp.StatusCode)
	}
	return codes
}

func TestRateLimit_LimitsSessions(t *testing.T) {
	assert.Equal(t, []int{200, 200, 429}, statuses(t, "user:123", 3))
}

func TestRateLimit_SkipsBackendService(t *testing.T) {
	assert.Equal(t, []int{200, 200, 200, 200}, statuses(t, "service:taxi-back-go", 4))
}
