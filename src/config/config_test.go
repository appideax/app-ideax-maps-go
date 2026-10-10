package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func clearEnv() {
	keys := []string{
		"APP_ENV", "PORT", "LOG_LEVEL", "APP_DOMAIN", "CORS_ALLOWED_ORIGINS",
		"JWT_SECRET", "MAP_TOKEN_TTL", "MAP_COOKIE_NAME", "MAP_COOKIE_DOMAIN",
		"UPSTREAM_TILES_URL", "UPSTREAM_GLYPHS_URL", "UPSTREAM_PHOTON_URL",
		"UPSTREAM_PELIAS_URL", "UPSTREAM_VALHALLA_URL", "RATE_LIMIT_PER_MINUTE",
		"TRUSTED_PROXIES", "CACHE_TTL_TILES", "CACHE_TTL_GLYPHS", "CACHE_TTL_GEOCODE", "CACHE_TTL_VALHALLA",
	}
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

func setValidEnv() {
	os.Setenv("APP_DOMAIN", "maps-api.appideax.com")
	os.Setenv("MAP_COOKIE_DOMAIN", ".appideax.com")
	os.Setenv("UPSTREAM_TILES_URL", "https://maps.example.com/ecuador")
	os.Setenv("UPSTREAM_GLYPHS_URL", "https://maps.example.com/font")
	os.Setenv("UPSTREAM_PHOTON_URL", "https://photon.example.com")
	os.Setenv("UPSTREAM_PELIAS_URL", "https://pelias.example.com")
	os.Setenv("UPSTREAM_VALHALLA_URL", "https://valhalla.example.com")
	os.Setenv("JWT_SECRET", "this-is-a-secret-key-for-tests-only-32")
}

func TestLoad_Defaults(t *testing.T) {
	clearEnv()
	setValidEnv()

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "3000", cfg.Server.Port)
	assert.Equal(t, "maps-api.appideax.com", cfg.App.Domain)
	assert.Equal(t, []string{"*"}, cfg.App.CORSAllowedOrigins)
	assert.Equal(t, "maps_session", cfg.Map.CookieName)
	assert.Equal(t, 120, cfg.RateLimit.PerMinute)
}

func TestLoad_MissingRequired(t *testing.T) {
	clearEnv()

	_, err := Load()
	require.Error(t, err)

	assert.Contains(t, err.Error(), "APP_DOMAIN")
	assert.Contains(t, err.Error(), "MAP_COOKIE_DOMAIN")
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_InvalidAppEnv(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("APP_ENV", "invalid")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "APP_ENV")
}

func TestLoad_InvalidPort(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("PORT", "abc")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PORT")
}

func TestLoad_InvalidDuration(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("CACHE_TTL_TILES", "not-a-duration")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CACHE_TTL_TILES")
}

func TestLoad_ProductionRequiresCORS(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("CORS_ALLOWED_ORIGINS", "")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_ORIGINS")
}

func TestLoad_ProductionRequiresStrongJWT(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "short")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}

func TestLoad_ProductionRejectsPlaceholderJWT(t *testing.T) {
	clearEnv()
	setValidEnv()
	os.Setenv("APP_ENV", "production")
	os.Setenv("JWT_SECRET", "change-me-in-production")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_SECRET")
}
