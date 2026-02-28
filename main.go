package main

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/handlers"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"

	"github.com/gofiber/fiber/v3"
)

func main() {
	pendingFilesManager := utils.NewPendingFilesManager()
	pendingFilesManager.StartCleanup()
	utils.FlushTempFiles()
	// utils.StartFileCleanup()

	env := config.LoadConfig()
	flake := utils.NewFlake()
	jwt := security.NewJWTService(env.JwtSecret)

	app := fiber.New(fiber.Config{
		StreamRequestBody: true,
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Nerimity CDN Online.")
	})

	contentHandler := handlers.NewContentHandler(&handlers.ContentHandler{Env: env})
	uploadHandler := handlers.NewUploadHandler(&handlers.UploadHandler{Env: env, Flake: flake, Jwt: jwt, PendingFilesManager: pendingFilesManager})
	internalHandler := handlers.NewInternalHandler(&handlers.InternalHandler{Env: env, Jwt: jwt})

	app.Get("/attachments/*", contentHandler.GetContent)
	app.Get("/emojis/*", contentHandler.GetContent)
	app.Get("/avatars/*", contentHandler.GetContent)
	app.Get("/profile_banners/*", contentHandler.GetContent)
	app.Get("/external-embed/*", contentHandler.GetContent)

	app.Post("/attachments/:groupId", uploadHandler.UploadFile)
	app.Post("/avatars/:groupId", uploadHandler.UploadFile)
	app.Post("/profile_banners/:groupId", uploadHandler.UploadFile)
	app.Post("/emojis", uploadHandler.UploadFile)

	app.Post("/internal/generate-token", internalHandler.GenerateToken)

	app.Listen(":" + env.Port)

}
