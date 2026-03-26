package utils

import "github.com/gofiber/fiber/v3"

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func Success(c fiber.Ctx, code int, message string, data any) error {
	return c.Status(code).JSON(Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Fail(c fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(Response{
		Success: false,
		Message: message,
		Error:   message,
	})
}
