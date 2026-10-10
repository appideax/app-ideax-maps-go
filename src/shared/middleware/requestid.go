package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/google/uuid"
)

const RequestIDKey = "requestid"

func RequestID() fiber.Handler {
	return requestid.New(requestid.Config{
		Header:     "X-Request-ID",
		ContextKey: RequestIDKey,
		Generator:  func() string { return uuid.NewString() },
	})
}
