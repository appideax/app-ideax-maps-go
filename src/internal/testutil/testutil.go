package testutil

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/cache"
	"app-ideax-maps-go/shared/middleware"
	"app-ideax-maps-go/shared/proxy"
)

func NewTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Env:                "development",
			LogLevel:           slog.LevelDebug,
			CORSAllowedOrigins: []string{"*"},
			TrustedProxies:     []string{"127.0.0.1"},
		},
		Server: config.ServerConfig{Port: "3002"},
		Auth: config.AuthConfig{
			JWTSecret:  "test-secret-key-with-at-least-32-characters",
			CookieName: "maps_session",
		},
		Upstream: config.UpstreamConfig{
			TilesURL:    "https://maps.example.com/ecuador",
			GlyphsURL:   "https://maps.example.com/font",
			PhotonURL:   "https://photon.example.com",
			PeliasURL:   "https://pelias.example.com",
			ValhallaURL: "https://valhalla.example.com",
			Timeout:     5 * time.Second,
		},
		RateLimit: config.RateLimitConfig{
			TilesPerMinute:   1200,
			GeocodePerMinute: 60,
			RoutingPerMinute: 60,
		},
		Cache: config.CacheConfig{
			MaxBytes:   8 * 1024 * 1024,
			TTLTiles:   24 * time.Hour,
			TTLGlyphs:  720 * time.Hour,
			TTLGeocode: 5 * time.Minute,
		},
	}
}

func NewTestInfrastructure(cfg *config.Config) *app.Infrastructure {
	if cfg == nil {
		cfg = NewTestConfig()
	}
	return &app.Infrastructure{
		Config: cfg,
		Cache:  cache.NewMemoryCache(cfg.Cache.MaxBytes),
		HTTP:   proxy.NewHTTPClient(cfg.Upstream.Timeout),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

// Token firma un token de mapas valido para cfg.
func Token(t *testing.T, cfg *config.Config) string {
	t.Helper()
	token, err := middleware.BuildToken(cfg.Auth.JWTSecret, time.Hour, "user-123", "company-456")
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	return token
}
