package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"app-ideax-maps-go/app"

	"github.com/gofiber/fiber/v2"
)

const UserAgent = "AppIdeaxMapsProxy/1.0"

var sensitiveHeaders = map[string]struct{}{
	"Authorization": {},
	"Cookie":        {},
	"Set-Cookie":    {},
	"X-Api-Key":     {},
	"X-Api-Secret":  {},
}

type Options struct {
	UpstreamBaseURL    string
	TargetPath         string
	AllowedParams      []string
	CacheKey           string
	CacheTTL           time.Duration
	Method             string
	Body               []byte
	ContentType        string
	DefaultContentType string
	QueryParams        url.Values
	// Transform recibe el body crudo del upstream y devuelve el body que se
	// enviará al cliente. Se ejecuta antes de guardar en caché.
	Transform func([]byte) ([]byte, error)
}

func Forward(c *fiber.Ctx, infra *app.Infrastructure, opts Options) error {
	if opts.Method == "" {
		opts.Method = c.Method()
	}

	upstreamURL, err := buildUpstreamURL(opts.UpstreamBaseURL, opts.TargetPath, opts.QueryParams, opts.AllowedParams)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid upstream URL")
	}

	if opts.CacheKey != "" && opts.CacheTTL > 0 {
		if cached, ok := infra.Cache.Get(opts.CacheKey); ok {
			setCacheHeaders(c, opts.CacheTTL)
			if opts.DefaultContentType != "" {
				c.Set("Content-Type", opts.DefaultContentType)
			}
			return c.Status(fiber.StatusOK).Send(cached)
		}
	}

	body := opts.Body
	if body == nil && opts.Method != http.MethodGet && opts.Method != http.MethodHead {
		body = c.Body()
	}

	req, err := http.NewRequestWithContext(c.UserContext(), opts.Method, upstreamURL, bytes.NewReader(body))
	if err != nil {
		if infra.Logger != nil {
			infra.Logger.Error("proxy: failed to build upstream request", "error", err)
		}
		return fiber.NewError(fiber.StatusInternalServerError, "proxy error")
	}

	req.Header.Set("User-Agent", UserAgent)
	if ae := c.Get("Accept-Encoding"); ae != "" {
		req.Header.Set("Accept-Encoding", ae)
	}
	if opts.ContentType != "" {
		req.Header.Set("Content-Type", opts.ContentType)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		if infra.Logger != nil {
			infra.Logger.Error("proxy: upstream request failed", "url", upstreamURL, "error", err)
		}
		return fiber.NewError(fiber.StatusBadGateway, "upstream unavailable")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if infra.Logger != nil {
			infra.Logger.Error("proxy: failed to read upstream response", "url", upstreamURL, "error", err)
		}
		return fiber.NewError(fiber.StatusBadGateway, "upstream error")
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 && opts.Transform != nil {
		transformed, err := opts.Transform(respBody)
		if err != nil {
			if infra.Logger != nil {
				infra.Logger.Error("proxy: transform failed", "url", upstreamURL, "error", err)
			}
			return fiber.NewError(fiber.StatusBadGateway, "upstream response invalid")
		}
		respBody = transformed
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 && len(respBody) > 0 && opts.CacheKey != "" && opts.CacheTTL > 0 {
		infra.Cache.Set(opts.CacheKey, respBody, opts.CacheTTL)
	}

	setResponseHeaders(c, resp)
	return c.Status(resp.StatusCode).Send(respBody)
}

func buildUpstreamURL(baseURL, targetPath string, params url.Values, allowedParams []string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + targetPath

	if params != nil {
		if len(allowedParams) > 0 {
			base.RawQuery = FilterQuery(params, allowedParams).Encode()
		} else {
			base.RawQuery = params.Encode()
		}
	}

	return base.String(), nil
}

func setCacheHeaders(c *fiber.Ctx, ttl time.Duration) {
	c.Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(ttl.Seconds())))
}

func setResponseHeaders(c *fiber.Ctx, resp *http.Response) {
	for key, values := range resp.Header {
		if _, sensitive := sensitiveHeaders[key]; sensitive {
			continue
		}
		for _, value := range values {
			c.Set(key, value)
		}
	}
}

func CacheKey(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	sort.Strings(clean)
	return strings.Join(clean, "|")
}

func QueryParams(c *fiber.Ctx) url.Values {
	params := make(url.Values)
	c.Request().URI().QueryArgs().VisitAll(func(key, value []byte) {
		params.Add(string(key), string(value))
	})
	return params
}

func FilterQuery(params url.Values, allowed []string) url.Values {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, p := range allowed {
		allowedSet[p] = struct{}{}
	}
	filtered := make(url.Values)
	for key, values := range params {
		if _, ok := allowedSet[key]; ok {
			for _, v := range values {
				filtered.Add(key, v)
			}
		}
	}
	return filtered
}

func CopyRequestHeaders(dst http.Header, src *fiber.Ctx) {
	for key, values := range src.GetReqHeaders() {
		if _, sensitive := sensitiveHeaders[key]; sensitive {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func LogProxyError(logger *slog.Logger, msg string, attrs ...any) {
	if logger != nil {
		logger.Error(msg, attrs...)
	}
}
