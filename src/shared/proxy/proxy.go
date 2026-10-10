package proxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"app-ideax-maps-go/app"
	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
)

const (
	UserAgent = "AppIdeaxMapsProxy/1.0"

	ContentTypeJSON     = "application/json; charset=utf-8"
	ContentTypeProtobuf = "application/x-protobuf"

	// Tope de lo que se lee de un servicio de mapas en una respuesta.
	maxUpstreamBody = 16 * 1024 * 1024
)

type Options struct {
	UpstreamBaseURL string
	TargetPath      string
	Query           url.Values
	// Method y Body: por defecto GET sin cuerpo.
	Method string
	Body   []byte
	// ContentType es el de la respuesta al cliente.
	ContentType string
	// Con CacheKey y CacheTTL la respuesta se guarda y se reutiliza.
	CacheKey string
	CacheTTL time.Duration
	// Transform recibe el body crudo del upstream y devuelve el que se envia al
	// cliente. Se ejecuta antes de guardar en cache.
	Transform func([]byte) ([]byte, error)
}

// NewHTTPClient arma el cliente compartido hacia los servicios de mapas. No se
// fija Accept-Encoding: asi Go descomprime solo y el cache guarda el contenido
// plano; la compresion hacia el cliente la hace el middleware de Fiber.
func NewHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 32
	return &http.Client{Timeout: timeout, Transport: transport}
}

// Forward pide el recurso al servicio de mapas y lo devuelve al cliente. Del
// request del cliente no se reenvia ninguna cabecera, y de la respuesta del
// upstream solo salen el estado y el cuerpo.
func Forward(c *fiber.Ctx, infra *app.Infrastructure, opts Options) error {
	cacheable := opts.CacheKey != "" && opts.CacheTTL > 0

	if cacheable {
		if cached, ok := infra.Cache.Get(opts.CacheKey); ok {
			return send(c, fiber.StatusOK, cached, opts)
		}
	}

	method := opts.Method
	if method == "" {
		method = http.MethodGet
	}
	upstreamURL := opts.UpstreamBaseURL + opts.TargetPath
	if len(opts.Query) > 0 {
		upstreamURL += "?" + opts.Query.Encode()
	}

	req, err := http.NewRequestWithContext(c.UserContext(), method, upstreamURL, bytes.NewReader(opts.Body))
	if err != nil {
		infra.Logger.Error("proxy: failed to build upstream request", "url", upstreamURL, "error", err)
		return response.Error(c, "proxy error", fiber.StatusInternalServerError)
	}
	req.Header.Set("User-Agent", UserAgent)
	if len(opts.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := infra.HTTP.Do(req)
	if err != nil {
		infra.Logger.Error("proxy: upstream request failed", "url", upstreamURL, "error", err)
		return response.Error(c, "upstream unavailable", fiber.StatusBadGateway)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxUpstreamBody+1))
	if err != nil || len(body) > maxUpstreamBody {
		infra.Logger.Error("proxy: failed to read upstream response", "url", upstreamURL, "error", err, "bytes", len(body))
		return response.Error(c, "upstream error", fiber.StatusBadGateway)
	}

	switch {
	case resp.StatusCode >= 500:
		infra.Logger.Error("proxy: upstream error", "url", upstreamURL, "status", resp.StatusCode)
		return response.Error(c, "upstream error", fiber.StatusBadGateway)
	case resp.StatusCode >= 300:
		// Errores del pedido (4xx): el cliente recibe el detalle del servicio.
		c.Set(fiber.HeaderCacheControl, "no-store")
		c.Set(fiber.HeaderContentType, ContentTypeJSON)
		return c.Status(resp.StatusCode).Send(body)
	case len(body) == 0:
		// Tesela vacia: Martin responde 204.
		return c.SendStatus(fiber.StatusNoContent)
	}

	if opts.Transform != nil {
		body, err = opts.Transform(body)
		if err != nil {
			infra.Logger.Error("proxy: transform failed", "url", upstreamURL, "error", err)
			return response.Error(c, "upstream response invalid", fiber.StatusBadGateway)
		}
	}

	if cacheable {
		infra.Cache.Set(opts.CacheKey, body, opts.CacheTTL)
	}

	return send(c, fiber.StatusOK, body, opts)
}

func send(c *fiber.Ctx, status int, body []byte, opts Options) error {
	// "private": la respuesta depende de la sesion, solo la guarda el navegador.
	if opts.CacheTTL > 0 {
		c.Set(fiber.HeaderCacheControl, fmt.Sprintf("private, max-age=%d", int(opts.CacheTTL.Seconds())))
	} else {
		c.Set(fiber.HeaderCacheControl, "no-store")
	}
	c.Set(fiber.HeaderContentType, opts.ContentType)
	return c.Status(status).Send(body)
}
