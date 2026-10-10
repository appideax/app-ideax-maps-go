package response

import (
	"github.com/gofiber/fiber/v2"
)

const MessageLocalKey = "response_message"

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func Success(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *fiber.Ctx, message string, status int) error {
	c.Locals(MessageLocalKey, message)
	return c.Status(status).JSON(APIResponse{
		Success: false,
		Message: message,
	})
}
