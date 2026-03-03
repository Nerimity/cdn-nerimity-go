package utils

import "github.com/gofiber/fiber/v3"

func SendError(c fiber.Ctx, status int, message string) error {
	c.Status(status).JSON(fiber.Map{
		"message": message,
	})
	return fiber.NewError(status, message)
}
