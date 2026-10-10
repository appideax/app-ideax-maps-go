package testutil

import (
	"log/slog"
	"time"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/cache"
)

func NewTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Env:                "development",
			LogLevel:           slog.LevelDebug,
			Domain:             "maps-api.appideax.com",
			CORSAllowedOrigins: []string{"*"},
			TrustedProxies:     []string{"127.0.0.1"},
		},
		Server: config.ServerConfig{Port: "3000"},
		JWT: config.JWTConfig{
			Secret:      "test-secret-key-with-at-least-32-characters",
			MapTokenTTL: time.Hour,
		},
		Map: config.MapConfig{
			CookieName:   "maps_session",
			CookieDomain: ".appideax.com",
		},
		Upstream: config.UpstreamConfig{
			TilesURL:    "https://maps.example.com/ecuador",
			GlyphsURL:   "https://maps.example.com/font",
			PhotonURL:   "https://photon.example.com",
			PeliasURL:   "https://pelias.example.com",
			ValhallaURL: "https://valhalla.example.com",
		},
		RateLimit: config.RateLimitConfig{PerMinute: 120},
		Cache: config.CacheConfig{
			TTLTiles:    24 * time.Hour,
			TTLGlyphs:   720 * time.Hour,
			TTLGeocode:  5 * time.Minute,
			TTLValhalla: 5 * time.Minute,
		},
	}
}

func NewTestInfrastructure(cfg *config.Config) *app.Infrastructure {
	if cfg == nil {
		cfg = NewTestConfig()
	}
	return &app.Infrastructure{
		Config: cfg,
		Cache:  cache.NewMemoryCache(time.Hour),
		Logger: slog.Default(),
	}
}
