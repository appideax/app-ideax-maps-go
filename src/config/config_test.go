package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var envKeys = []string{
	"APP_ENV", "PORT", "LOG_LEVEL", "CORS_ALLOWED_ORIGINS", "TRUSTED_PROXIES",
	"MAPS_JWT_SECRET", "MAPS_COOKIE_NAME",
	"UPSTREAM_TILES_URL", "UPSTREAM_GLYPHS_URL",
	"UPSTREAM_PELIAS_URL", "UPSTREAM_VALHALLA_URL", "UPSTREAM_TIMEOUT",
	"RATE_LIMIT_TILES_PER_MINUTE", "RATE_LIMIT_GEOCODE_PER_MINUTE", "RATE_LIMIT_ROUTING_PER_MINUTE",
	"CACHE_MAX_MB", "CACHE_TTL_TILES", "CACHE_TTL_GLYPHS", "CACHE_TTL_GEOCODE",
}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range envKeys {
		t.Setenv(k, "")
	}
}

func setValidEnv(t *testing.T) {
	t.Helper()
	clearEnv(t)
	t.Setenv("UPSTREAM_TILES_URL", "https://maps.example.com/ecuador/")
	t.Setenv("UPSTREAM_GLYPHS_URL", "https://maps.example.com/font")
	t.Setenv("UPSTREAM_PELIAS_URL", "https://pelias.example.com")
	t.Setenv("UPSTREAM_VALHALLA_URL", "https://valhalla.example.com")
	t.Setenv("MAPS_JWT_SECRET", "this-is-a-secret-key-for-tests-only-32")
}

func TestLoad_Defaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.App.Env)
	assert.Equal(t, "3002", cfg.Server.Port)
	assert.Equal(t, []string{"*"}, cfg.App.CORSAllowedOrigins)
	assert.Equal(t, "maps_session", cfg.Auth.CookieName)
	assert.Equal(t, "https://maps.example.com/ecuador", cfg.Upstream.TilesURL)
	assert.Equal(t, 10*time.Second, cfg.Upstream.Timeout)
	assert.Equal(t, 1200, cfg.RateLimit.TilesPerMinute)
	assert.Equal(t, 60, cfg.RateLimit.GeocodePerMinute)
	assert.Equal(t, 60, cfg.RateLimit.RoutingPerMinute)
	assert.Equal(t, int64(256*1024*1024), cfg.Cache.MaxBytes)
}

func TestLoad_MissingRequired(t *testing.T) {
	clearEnv(t)

	_, err := Load()
	require.Error(t, err)

	assert.Contains(t, err.Error(), "MAPS_JWT_SECRET")
	assert.Contains(t, err.Error(), "UPSTREAM_TILES_URL")
	assert.Contains(t, err.Error(), "UPSTREAM_VALHALLA_URL")
}

func TestLoad_InvalidValues(t *testing.T) {
	cases := map[string]string{
		"APP_ENV":                     "invalid",
		"PORT":                        "abc",
		"CACHE_TTL_TILES":             "not-a-duration",
		"UPSTREAM_PELIAS_URL":         "pelias.example.com",
		"RATE_LIMIT_TILES_PER_MINUTE": "0",
		"CACHE_MAX_MB":                "-1",
	}
	for key, value := range cases {
		t.Run(key, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(key, value)

			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), key)
		})
	}
}

func TestLoad_ProductionRequiresCORS(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENV", "production")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_ORIGINS")
}

func TestLoad_ProductionRejectsWildcardCORS(t *testing.T) {
	setValidEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://appideax.com,*")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_ORIGINS")
}

func TestLoad_ProductionRequiresStrongSecret(t *testing.T) {
	for _, secret := range []string{"short", "change-me-in-production"} {
		setValidEnv(t)
		t.Setenv("APP_ENV", "production")
		t.Setenv("CORS_ALLOWED_ORIGINS", "https://appideax.com")
		t.Setenv("MAPS_JWT_SECRET", secret)

		_, err := Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "MAPS_JWT_SECRET")
	}
}
