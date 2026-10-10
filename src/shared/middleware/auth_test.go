package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app-ideax-maps-go/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAuthMiddleware(secret, cookieName string) fiber.Handler {
	cfg := config.JWTConfig{Secret: secret}
	return Auth(cfg, cookieName)
}

func TestAuth_BearerValid(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware(secret, "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user_id":    c.Locals("user_id"),
			"company_id": c.Locals("company_id"),
			"roles":      c.Locals("roles"),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_QueryTokenValid(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, time.Hour, "", "", "ride-789", []string{})
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware(secret, "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"ride_id": c.Locals("ride_id")})
	})

	req := httptest.NewRequest(http.MethodGet, "/test?token="+token, nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_CookieValid(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, time.Hour, "user-123", "company-456", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware(secret, "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "maps_session", Value: token})
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_MissingToken(t *testing.T) {
	app := fiber.New()
	app.Use(testAuthMiddleware("test-secret-key-with-at-least-32-characters", "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_ExpiredToken(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, -time.Hour, "user-123", "", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware(secret, "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_InvalidSignature(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, time.Hour, "user-123", "", "", []string{"admin"})
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware("different-secret-key-with-at-least-32-characters", "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	discardBody(t, resp)
}

func TestAuth_MissingUserAndRide(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	claims := MapClaims{
		CompanyID: "company-456",
		Roles:     []string{"admin"},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    expectedIssuer,
			Audience:  jwt.ClaimStrings{expectedAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)

	app := fiber.New()
	app.Use(testAuthMiddleware(secret, "maps_session"))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	discardBody(t, resp)
}

func TestBuildToken(t *testing.T) {
	secret := "test-secret-key-with-at-least-32-characters"
	token, err := BuildToken(secret, time.Hour, "user-123", "company-456", "ride-789", []string{"admin"})
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := parseAndValidate(token, secret)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "company-456", claims.CompanyID)
	assert.Equal(t, "ride-789", claims.RideID)
	assert.Equal(t, []string{"admin"}, claims.Roles)
}

func discardBody(t *testing.T, resp *http.Response) {
	t.Helper()
	_, err := io.Copy(io.Discard, resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
}
