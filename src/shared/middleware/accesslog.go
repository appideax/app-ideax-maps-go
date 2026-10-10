package middleware

import (
	"errors"
	"log/slog"
	"time"

	"app-ideax-maps-go/shared/response"

	"github.com/gofiber/fiber/v2"
)

func AccessLog(logger *slog.Logger) fiber.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *fiber.Ctx) error {
		if c.Path() == "/healthz" {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			status = fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				status = fiberErr.Code
			}
		}

		attrs := []any{
			"method", c.Method(),
			"path", c.Path(),
			"status", status,
			"latencyMs", time.Since(start).Milliseconds(),
			"ip", ClientIP(c),
		}
		if rid, ok := c.Locals(RequestIDKey).(string); ok && rid != "" {
			attrs = append(attrs, "requestId", rid)
		}
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			attrs = append(attrs, "userId", uid)
		}
		if cid, ok := c.Locals("company_id").(string); ok && cid != "" {
			attrs = append(attrs, "companyId", cid)
		}
		if rID, ok := c.Locals("ride_id").(string); ok && rID != "" {
			attrs = append(attrs, "rideId", rID)
		}
		if err != nil {
			attrs = append(attrs, "error", err.Error())
		}
		if status >= 400 {
			if msg, ok := c.Locals(response.MessageLocalKey).(string); ok && msg != "" {
				attrs = append(attrs, "message", msg)
			}
		}

		switch {
		case status >= 500:
			logger.Error("http request", attrs...)
		case status >= 400:
			logger.Warn("http request", attrs...)
		default:
			logger.Info("http request", attrs...)
		}

		return err
	}
}
