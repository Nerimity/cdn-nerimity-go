package main

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/handlers"
	"cdn_nerimity_go/utils"

	"github.com/gofiber/fiber/v3"
)

func main() {
	// utils.FlushTempFiles()
	utils.StartFileCleanup()
	env := config.LoadConfig()
	flake := utils.NewFlake()

	app := fiber.New(fiber.Config{
		StreamRequestBody: true,
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Nerimity CDN Online.")
	})

	contentHandler := handlers.NewContentHandler(&handlers.ContentHandler{Env: env})
	uploadHandler := handlers.NewUploadHandler(&handlers.UploadHandler{Env: env, Flake: flake})

	app.Get("/attachments/*", contentHandler.GetContent)
	app.Get("/emojis/*", contentHandler.GetContent)
	app.Get("/avatars/*", contentHandler.GetContent)
	app.Get("/profile_banners/*", contentHandler.GetContent)
	app.Get("/external-embed/*", contentHandler.GetContent)

	app.Post("/attachments/*", uploadHandler.UploadFile)

	app.Listen(":" + config.LoadConfig().Port)

}
