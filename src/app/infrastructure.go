package app

import (
	"log/slog"
	"net/http"

	"app-ideax-maps-go/config"
	"app-ideax-maps-go/shared/cache"
)

type Infrastructure struct {
	Config *config.Config
	Cache  cache.Cache
	// HTTP es el cliente compartido hacia los servicios de mapas.
	HTTP   *http.Client
	Logger *slog.Logger
}
