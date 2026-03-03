package main

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/database"
	"cdn_nerimity_go/handlers"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"path/filepath"
	"runtime"

	"github.com/cshum/vipsgen/vips"
	"github.com/gofiber/fiber/v3"
)

func main() {
	env := config.LoadConfig()

	_, filename, _, _ := runtime.Caller(0)
	env.ProjectRoot = filepath.Dir(filename)

	pendingFilesManager := utils.NewPendingFilesManager()
	pendingFilesManager.StartCleanup()
	utils.FlushTempFiles()
	// utils.StartFileCleanup()

	flake := utils.NewFlake()
	jwt := security.NewJWTService(env.JwtSecret)
	database := database.NewDatabaseService(env.DatabaseUrl)
	utils.StartDeleteExpiredFiles(database)

	vips.Startup(nil)
	defer vips.Shutdown()

	app := fiber.New(fiber.Config{
		StreamRequestBody: true,
	})

	app.Get("/", func(c fiber.Ctx) error {
		return c.SendString("Nerimity CDN Online.")
	})

	contentHandler := handlers.NewContentHandler(&handlers.ContentHandler{Env: env})
	uploadHandler := handlers.NewUploadHandler(&handlers.UploadHandler{Env: env, Flake: flake, Jwt: jwt, PendingFilesManager: pendingFilesManager})
	internalHandler := handlers.NewInternalHandler(&handlers.InternalHandler{Env: env, Jwt: jwt, PendingFileManager: pendingFilesManager, Database: database})
	proxyHandler := handlers.NewProxyHandler(&handlers.ProxyHandler{Env: env})

	app.Get("/attachments/*", contentHandler.GetContent)
	app.Get("/emojis/*", contentHandler.GetContent)
	app.Get("/avatars/*", contentHandler.GetContent)
	app.Get("/profile_banners/*", contentHandler.GetContent)
	app.Get("/external-embed/*", contentHandler.GetContent)

	app.Post("/attachments/:groupId", uploadHandler.UploadFile)
	app.Post("/avatars/:groupId", uploadHandler.UploadFile)
	app.Post("/profile_banners/:groupId", uploadHandler.UploadFile)
	app.Post("/emojis", uploadHandler.UploadFile)

	app.Get("/proxy-dimensions", proxyHandler.GetImageDimensions)
	app.Get("/proxy/:imageUrl/:filename", proxyHandler.GetProxy)

	app.Post("/internal/generate-token", internalHandler.GenerateToken)
	app.Post("/internal/verify-file", internalHandler.VerifyFile)
	app.Delete("/internal/batch", internalHandler.DeleteByFileIds)
	app.Delete("/internal/attachments/:groupId/batch", internalHandler.DeleteAttachmentsByGroupId)
	app.Delete("/internal/", internalHandler.DeleteFile)

	app.Listen(":" + env.Port)

}
