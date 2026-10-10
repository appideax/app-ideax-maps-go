package app

import (
	"log/slog"

	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/cache"
)

type Infrastructure struct {
	Config *config.Config
	Cache  cache.Cache
	Logger *slog.Logger
}
