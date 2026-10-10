package middleware

import (
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

const testSecret = "test-secret-key-with-at-least-32-characters"

func authApp() *fiber.App {
	app := fiber.New()
	app.Use(Auth(config.AuthConfig{JWTSecret: testSecret, CookieName: "maps_session"}))
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"subject": c.Locals(SubjectLocalKey),
			"company": c.Locals(CompanyLocalKey),
		})
	})
	return app
}

func authStatus(t *testing.T, prepare func(*http.Request)) int {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	prepare(req)
	resp, err := authApp().Test(req, -1)
	require.NoError(t, err)
	defer resp.Body.Close()
	return resp.StatusCode
}

func validToken(t *testing.T) string {
	t.Helper()
	token, err := BuildToken(testSecret, time.Hour, "user-123", "company-456")
	require.NoError(t, err)
	return token
}

// signClaims firma claims arbitrarios para probar los rechazos.
func signClaims(t *testing.T, secret string, mutate func(*MapClaims)) string {
	t.Helper()
	claims := &MapClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Audience:  jwt.ClaimStrings{TokenAudience},
			Subject:   "user-123",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	mutate(claims)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	require.NoError(t, err)
	return token
}

func TestAuth_BearerValid(t *testing.T) {
	token := validToken(t)
	status := authStatus(t, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) })
	assert.Equal(t, http.StatusOK, status)
}

func TestAuth_CookieValid(t *testing.T) {
	token := validToken(t)
	status := authStatus(t, func(r *http.Request) {
		r.AddCookie(&http.Cookie{Name: "maps_session", Value: token})
	})
	assert.Equal(t, http.StatusOK, status)
}

func TestAuth_ValidCookieWinsOverStaleBearer(t *testing.T) {
	token := validToken(t)
	status := authStatus(t, func(r *http.Request) {
		r.Header.Set("Authorization", "Bearer stale")
		r.AddCookie(&http.Cookie{Name: "maps_session", Value: token})
	})
	assert.Equal(t, http.StatusOK, status)
}

func TestAuth_MissingToken(t *testing.T) {
	assert.Equal(t, http.StatusUnauthorized, authStatus(t, func(*http.Request) {}))
}

func TestAuth_QueryTokenNotAccepted(t *testing.T) {
	token := validToken(t)
	status := authStatus(t, func(r *http.Request) { r.URL.RawQuery = "token=" + token })
	assert.Equal(t, http.StatusUnauthorized, status)
}

func TestAuth_RejectsInvalidTokens(t *testing.T) {
	cases := map[string]string{
		"wrong secret":   signClaims(t, "another-secret-key-with-at-least-32-chars", func(*MapClaims) {}),
		"wrong issuer":   signClaims(t, testSecret, func(c *MapClaims) { c.Issuer = "someone-else" }),
		"wrong audience": signClaims(t, testSecret, func(c *MapClaims) { c.Audience = jwt.ClaimStrings{"api"} }),
		"no subject":     signClaims(t, testSecret, func(c *MapClaims) { c.Subject = "" }),
		"expired":        signClaims(t, testSecret, func(c *MapClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute)) }),
		"no expiration":  signClaims(t, testSecret, func(c *MapClaims) { c.ExpiresAt = nil }),
		"garbage":        "not-a-jwt",
	}
	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			status := authStatus(t, func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) })
			assert.Equal(t, http.StatusUnauthorized, status)
		})
	}
}
