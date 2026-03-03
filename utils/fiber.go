package utils

import "github.com/gofiber/fiber/v3"

func SendError(c fiber.Ctx, code int, message string) error {
	return fiber.NewError(code, message)
}
