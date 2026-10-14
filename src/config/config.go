package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	Auth      AuthConfig
	Upstream  UpstreamConfig
	RateLimit RateLimitConfig
	Cache     CacheConfig
}

type AppConfig struct {
	Env                string
	LogLevel           slog.Level
	CORSAllowedOrigins []string
	TrustedProxies     []string
}

type ServerConfig struct {
	Port string
}

// AuthConfig valida los tokens de mapas que firma taxi-back-go. El secreto es
// el mismo MAPS_JWT_SECRET de ese servicio y no es su JWT_SECRET.
type AuthConfig struct {
	JWTSecret  string
	CookieName string
}

type UpstreamConfig struct {
	TilesURL    string
	GlyphsURL   string
	PeliasURL   string
	ValhallaURL string
	Timeout     time.Duration
}

// RateLimitConfig: peticiones por minuto de cada sesion de mapas. Las teselas
// van aparte porque un solo movimiento del mapa pide decenas.
type RateLimitConfig struct {
	TilesPerMinute   int
	GeocodePerMinute int
	RoutingPerMinute int
}

type CacheConfig struct {
	MaxBytes   int64
	TTLTiles   time.Duration
	TTLGlyphs  time.Duration
	TTLGeocode time.Duration
}

func Load() (*Config, error) {
	var errs []error

	appEnv := strings.TrimSpace(os.Getenv("APP_ENV"))
	if appEnv == "" {
		appEnv = "development"
	}
	switch appEnv {
	case "development", "staging", "production":
	default:
		errs = append(errs, fmt.Errorf("invalid value for APP_ENV: %q (must be development, staging or production)", appEnv))
	}

	port, err := envPort("PORT", "3002")
	if err != nil {
		errs = append(errs, err)
	}

	logLevel, err := parseLogLevel(os.Getenv("LOG_LEVEL"))
	if err != nil {
		errs = append(errs, err)
	}

	// Sin origenes configurados se acepta cualquiera, solo fuera de produccion.
	origins := parseList(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(origins) == 0 {
		if appEnv == "production" {
			errs = append(errs, errors.New("missing required environment variable: CORS_ALLOWED_ORIGINS (required when APP_ENV=production)"))
		} else {
			origins = []string{"*"}
		}
	}

	trustedProxies := parseList(os.Getenv("TRUSTED_PROXIES"))
	if len(trustedProxies) == 0 {
		trustedProxies = []string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	}

	jwtSecret := os.Getenv("MAPS_JWT_SECRET")
	if strings.TrimSpace(jwtSecret) == "" {
		errs = append(errs, errors.New("missing required environment variable: MAPS_JWT_SECRET"))
	}
	cookieName := envString("MAPS_COOKIE_NAME", "maps_session")

	upstream := UpstreamConfig{}
	for key, target := range map[string]*string{
		"UPSTREAM_TILES_URL":    &upstream.TilesURL,
		"UPSTREAM_GLYPHS_URL":   &upstream.GlyphsURL,
		"UPSTREAM_PELIAS_URL":   &upstream.PeliasURL,
		"UPSTREAM_VALHALLA_URL": &upstream.ValhallaURL,
	} {
		value, err := envURL(key)
		if err != nil {
			errs = append(errs, err)
		}
		*target = value
	}
	upstream.Timeout, err = envDuration("UPSTREAM_TIMEOUT", 10*time.Second)
	if err != nil {
		errs = append(errs, err)
	}

	rateLimit := RateLimitConfig{}
	for _, limit := range []struct {
		key          string
		defaultValue int
		target       *int
	}{
		{"RATE_LIMIT_TILES_PER_MINUTE", 1200, &rateLimit.TilesPerMinute},
		{"RATE_LIMIT_GEOCODE_PER_MINUTE", 60, &rateLimit.GeocodePerMinute},
		{"RATE_LIMIT_ROUTING_PER_MINUTE", 60, &rateLimit.RoutingPerMinute},
	} {
		value, err := envPositiveInt(limit.key, limit.defaultValue)
		if err != nil {
			errs = append(errs, err)
		}
		*limit.target = value
	}

	cacheMaxMB, err := envPositiveInt("CACHE_MAX_MB", 256)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLTiles, err := envDuration("CACHE_TTL_TILES", 24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLGlyphs, err := envDuration("CACHE_TTL_GLYPHS", 720*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}
	cacheTTLGeocode, err := envDuration("CACHE_TTL_GEOCODE", 24*time.Hour)
	if err != nil {
		errs = append(errs, err)
	}

	if appEnv == "production" {
		if slices.Contains(origins, "*") {
			errs = append(errs, errors.New(`invalid value for CORS_ALLOWED_ORIGINS: "*" is not allowed when APP_ENV=production`))
		}
		if len(jwtSecret) < 32 {
			errs = append(errs, fmt.Errorf("invalid value for MAPS_JWT_SECRET: must be at least 32 characters when APP_ENV=production (got %d)", len(jwtSecret)))
		}
		if isPlaceholderSecret(jwtSecret) {
			errs = append(errs, errors.New("invalid value for MAPS_JWT_SECRET: placeholder value is not allowed when APP_ENV=production"))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return &Config{
		App: AppConfig{
			Env:                appEnv,
			LogLevel:           logLevel,
			CORSAllowedOrigins: origins,
			TrustedProxies:     trustedProxies,
		},
		Server: ServerConfig{
			Port: port,
		},
		Auth: AuthConfig{
			JWTSecret:  jwtSecret,
			CookieName: cookieName,
		},
		Upstream:  upstream,
		RateLimit: rateLimit,
		Cache: CacheConfig{
			MaxBytes:   int64(cacheMaxMB) * 1024 * 1024,
			TTLTiles:   cacheTTLTiles,
			TTLGlyphs:  cacheTTLGlyphs,
			TTLGeocode: cacheTTLGeocode,
		},
	}, nil
}

func envString(key, defaultValue string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return defaultValue
}

func envPositiveInt(key string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid value for %s: %q must be an integer greater than 0", key, raw)
	}
	return n, nil
}

func envDuration(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("invalid value for %s: %q is not a valid duration (e.g. 5m, 24h, 720h)", key, raw)
	}
	return d, nil
}

func envPort(key, defaultValue string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue, nil
	}
	if n, err := strconv.Atoi(raw); err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("invalid value for %s: %q is not a valid port", key, raw)
	}
	return raw, nil
}

// envURL exige una URL http(s) y la devuelve sin la barra final.
func envURL(key string) (string, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return "", fmt.Errorf("missing required environment variable: %s", key)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid value for %s: %q is not an http(s) URL", key, raw)
	}
	return strings.TrimRight(raw, "/"), nil
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

func parseList(raw string) []string {
	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func isPlaceholderSecret(secret string) bool {
	switch strings.ToLower(strings.TrimSpace(secret)) {
	case "your-secret-key-change-in-production", "changeme", "change-me-in-production", "secret", "123456789", "apikey123456789":
		return true
	default:
		return false
	}
}
