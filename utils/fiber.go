package utils

import "github.com/gofiber/fiber/v3"

func SendError(c fiber.Ctx, code int, message string) error {
	return fiber.NewError(code, message)
}

var allowedOrigins = map[string]bool{
	"https://nerimity.com":           true,
	"http://local.nerimity.com:3000": true,
	"https://latest.nerimity.com":    true,
	"https://flutter.nerimity.com":   true,
}

func SetCorsHeader(c fiber.Ctx) {
	origin := c.Get("Origin")

	if origin != "" && allowedOrigins[origin] {
		c.Set("Access-Control-Allow-Origin", origin)
	} else {
		c.Set("Access-Control-Allow-Origin", "https://nerimity.com")
	}
}
