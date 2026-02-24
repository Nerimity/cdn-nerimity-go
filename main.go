package main

import (
	"cdn_nerimity_go/handlers"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Nerimity CDN Online.")
	})

	app.Get("/avatars/*", handlers.GetContentHandler)

	app.Listen("127.0.0.1:3000")

}
