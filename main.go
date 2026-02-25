package main

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/handlers"

	"github.com/gofiber/fiber/v3"
)

func main() {
	env := config.LoadConfig()

	app := fiber.New(fiber.Config{
		StreamRequestBody: true,
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Nerimity CDN Online.")
	})

	contentHandler := handlers.NewContentHandler(env)
	uploadHandler := handlers.NewUploadHandler(env)

	app.Get("/attachments/*", contentHandler.GetContent)
	app.Get("/emojis/*", contentHandler.GetContent)
	app.Get("/avatars/*", contentHandler.GetContent)
	app.Get("/profile_banners/*", contentHandler.GetContent)
	app.Get("/external-embed/*", contentHandler.GetContent)

	app.Post("/attachments/*", uploadHandler.UploadFile)

	app.Listen(":" + config.LoadConfig().Port)

}
