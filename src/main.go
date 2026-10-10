package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/config"
	"app-ideax-maps-go/routes"
	"app-ideax-maps-go/shared/cache"
	"app-ideax-maps-go/shared/logging"
	"app-ideax-maps-go/shared/middleware"
	"app-ideax-maps-go/shared/proxy"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load("../.env")

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	appLogger, endBootLogs := logging.New(cfg.App.LogLevel)
	slog.SetDefault(appLogger)

	infra := &app.Infrastructure{
		Config: cfg,
		Cache:  cache.NewMemoryCache(cfg.Cache.MaxBytes),
		HTTP:   proxy.NewHTTPClient(cfg.Upstream.Timeout),
		Logger: appLogger,
	}

	fiberApp := fiber.New(fiber.Config{
		AppName:                 "app-ideax-maps-go",
		ReadTimeout:             30 * time.Second,
		WriteTimeout:            30 * time.Second,
		IdleTimeout:             60 * time.Second,
		BodyLimit:               1024 * 1024,
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          cfg.App.TrustedProxies,
	})

	middleware.SetTrustedProxies(cfg.App.TrustedProxies)

	fiberApp.Use(middleware.RequestID())
	fiberApp.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e interface{}) {
			attrs := []any{"error", e, "method", c.Method(), "path", c.Path()}
			if rid, ok := c.Locals(middleware.RequestIDKey).(string); ok && rid != "" {
				attrs = append(attrs, "requestId", rid)
			}
			if cfg.App.Env != "production" {
				attrs = append(attrs, "stack", string(debug.Stack()))
			}
			appLogger.Error("panic recovered", attrs...)
		},
	}))
	fiberApp.Use(middleware.AccessLog(appLogger))
	fiberApp.Use(middleware.CORS(cfg.App.CORSAllowedOrigins))
	// El cache guarda el contenido plano; aca se comprime segun lo que acepte el cliente.
	fiberApp.Use(compress.New(compress.Config{Level: compress.LevelBestSpeed}))

	routes.Register(fiberApp, infra)

	endBootLogs()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- fiberApp.Listen(":" + cfg.Server.Port)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	exitCode := 0
	select {
	case sig := <-quit:
		appLogger.Info("shutting down", "signal", sig.String())
	case err := <-listenErr:
		appLogger.Error("fiber server error, shutting down", "error", err)
		exitCode = 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := fiberApp.ShutdownWithContext(ctx); err != nil {
		slog.Error("fiber shutdown error", "error", err)
	}
	cancel()

	appLogger.Info("server stopped")
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
