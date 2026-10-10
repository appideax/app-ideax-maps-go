package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	JWT       JWTConfig
	Map       MapConfig
	Upstream  UpstreamConfig
	RateLimit RateLimitConfig
	Cache     CacheConfig
}

type AppConfig struct {
	Env                string
	LogLevel           slog.Level
	Domain             string
	CORSAllowedOrigins []string
	TrustedProxies     []string
}

type ServerConfig struct {
	Port string
}

type JWTConfig struct {
	Secret      string
	MapTokenTTL time.Duration
}

type MapConfig struct {
	CookieName   string
	CookieDomain string
}

type UpstreamConfig struct {
	TilesURL    string
	GlyphsURL   string
	PhotonURL   string
	PeliasURL   string
	ValhallaURL string
}

type RateLimitConfig struct {
	PerMinute int
}

type CacheConfig struct {
	TTLTiles    time.Duration
	TTLGlyphs   time.Duration
	TTLGeocode  time.Duration
	TTLValhalla time.Duration
}

func Load() (*Config, error) {
	var errs []error

	require := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			errs = append(errs, fmt.Errorf("missing required environment variable: %s", key))
		}
	}

	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "development"
	}
	switch appEnv {
	case "development", "staging", "production":
	default:
		errs = append(errs, fmt.Errorf("invalid value for APP_ENV: %q (must be development, staging or production)", appEnv))
	}

	port, err := envPort("PORT", "3000")
	if err != nil {
		errs = append(errs, err)
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		errs = append(errs, err)
	}

	appDomain := strings.TrimSpace(os.Getenv("APP_DOMAIN"))
	require("APP_DOMAIN", appDomain)

	corsRaw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	origins := parseOrigins(corsRaw)
	if corsRaw == "" {
		if appEnv == "production" {
			errs = append(errs, errors.New("missing required environment variable: CORS_ALLOWED_ORIGINS (required when APP_ENV=production)"))
		} else {
			origins = []string{"*"}
		}
	}

	trustedProxiesRaw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if trustedProxiesRaw == "" {
		trustedProxiesRaw = "127.0.0.1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
	}
	trustedProxies := make([]string, 0, 4)
	for _, p := range strings.Split(trustedProxiesRaw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			trustedProxies = append(trustedProxies, p)
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	require("JWT_SECRET", jwtSecret)

	mapTokenTTL, err := envDuration("MAP_TOKEN_TTL", 5*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	mapCookieName := getEnvOrDefault("MAP_COOKIE_NAME", "maps_session")
	mapCookieDomain := strings.TrimSpace(os.Getenv("MAP_COOKIE_DOMAIN"))
	require("MAP_COOKIE_DOMAIN", mapCookieDomain)

	upstreamTilesURL := strings.TrimSpace(os.Getenv("UPSTREAM_TILES_URL"))
	upstreamGlyphsURL := strings.TrimSpace(os.Getenv("UPSTREAM_GLYPHS_URL"))
	upstreamPhotonURL := strings.TrimSpace(os.Getenv("UPSTREAM_PHOTON_URL"))
	upstreamPeliasURL := strings.TrimSpace(os.Getenv("UPSTREAM_PELIAS_URL"))
	upstreamValhallaURL := strings.TrimSpace(os.Getenv("UPSTREAM_VALHALLA_URL"))
	require("UPSTREAM_TILES_URL", upstreamTilesURL)
	require("UPSTREAM_GLYPHS_URL", upstreamGlyphsURL)
	require("UPSTREAM_PHOTON_URL", upstreamPhotonURL)
	require("UPSTREAM_PELIAS_URL", upstreamPeliasURL)
	require("UPSTREAM_VALHALLA_URL", upstreamValhallaURL)

	rateLimitPerMinute, err := envInt("RATE_LIMIT_PER_MINUTE", 120)
	if err != nil {
		errs = append(errs, err)
	}
	if rateLimitPerMinute <= 0 {
		errs = append(errs, errors.New("RATE_LIMIT_PER_MINUTE must be greater than 0"))
	}

	cacheTTLTiles, err := envDuration("CACHE_TTL_TILES", 24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLGlyphs, err := envDuration("CACHE_TTL_GLYPHS", 720*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLGeocode, err := envDuration("CACHE_TTL_GEOCODE", 5*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLValhalla, err := envDuration("CACHE_TTL_VALHALLA", 5*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}

	if appEnv == "production" {
		if containsWildcard(origins) {
			errs = append(errs, errors.New(`invalid value for CORS_ALLOWED_ORIGINS: "*" is not allowed when APP_ENV=production`))
		}
		if len(jwtSecret) < 32 {
			errs = append(errs, fmt.Errorf("invalid value for JWT_SECRET: must be at least 32 characters when APP_ENV=production (got %d)", len(jwtSecret)))
		}
		if isPlaceholderJWTSecret(jwtSecret) {
			errs = append(errs, errors.New("invalid value for JWT_SECRET: placeholder value is not allowed when APP_ENV=production"))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Config{
		App: AppConfig{
			Env:                appEnv,
			LogLevel:           logLevel,
			Domain:             appDomain,
			CORSAllowedOrigins: origins,
			TrustedProxies:     trustedProxies,
		},
		Server: ServerConfig{
			Port: port,
		},
		JWT: JWTConfig{
			Secret:      jwtSecret,
			MapTokenTTL: mapTokenTTL,
		},
		Map: MapConfig{
			CookieName:   mapCookieName,
			CookieDomain: mapCookieDomain,
		},
		Upstream: UpstreamConfig{
			TilesURL:    upstreamTilesURL,
			GlyphsURL:   upstreamGlyphsURL,
			PhotonURL:   upstreamPhotonURL,
			PeliasURL:   upstreamPeliasURL,
			ValhallaURL: upstreamValhallaURL,
		},
		RateLimit: RateLimitConfig{
			PerMinute: rateLimitPerMinute,
		},
		Cache: CacheConfig{
			TTLTiles:    cacheTTLTiles,
			TTLGlyphs:   cacheTTLGlyphs,
			TTLGeocode:  cacheTTLGeocode,
			TTLValhalla: cacheTTLValhalla,
		},
	}, nil
}

func envInt(key string, defaultValue int) (int, error) {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %q is not a valid integer", key, raw)
	}
	return n, nil
}

func envDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid value for %s: %q is not a valid duration (e.g. 5m, 24h, 720h)", key, raw)
	}
	return d, nil
}

func envPort(key, defaultValue string) (string, error) {
	raw := os.Getenv(key)
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	if _, err := strconv.Atoi(strings.TrimSpace(raw)); err != nil {
		return "", fmt.Errorf("invalid value for %s: %q is not a valid port", key, raw)
	}
	return strings.TrimSpace(raw), nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.TrimSpace(raw) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid value for LOG_LEVEL: %q (must be debug, info, warn or error)", raw)
	}
}

func parseOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func containsWildcard(origins []string) bool {
	for _, origin := range origins {
		if origin == "*" {
			return true
		}
	}
	return false
}

func isPlaceholderJWTSecret(secret string) bool {
	switch strings.ToLower(strings.TrimSpace(secret)) {
	case "your-secret-key-change-in-production", "changeme", "change-me-in-production", "secret", "123456789", "apikey123456789":
		return true
	default:
		return false
	}
}
