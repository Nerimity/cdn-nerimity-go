package handlers

import (
	"cdn_nerimity_go/utils"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

func GetContentHandler(c fiber.Ctx) error {
	urlPath := c.Path()

	decodedBasename, err := url.PathUnescape(path.Base(urlPath))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid encoding")
	}

	// Force path to be relative to prevent escaping to root
	relPath := path.Join(path.Dir(urlPath), decodedBasename)

	relPath = strings.TrimPrefix(relPath, "/")

	baseDir := "./public"
	finalPath := filepath.Join(baseDir, relPath)

	absBase, _ := filepath.Abs(baseDir)
	absFinal, err := filepath.Abs(finalPath)
	if err != nil || !strings.HasPrefix(absFinal, absBase) {
		return c.Status(fiber.StatusForbidden).End()
	}

	ext := filepath.Ext(finalPath)
	// Check file size
	info, err := os.Stat(finalPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).End()
	}
	if info.IsDir() {
		return c.Status(fiber.StatusNotFound).End()
	}
	fileSizeMB := info.Size() / (1024 * 1024) // size in MB
	var isImageFileSize = fileSizeMB <= 20

	if utils.IsImage(strings.ToLower(ext)) && isImageFileSize {
		imageType := c.Query("type")
		size := c.Query("size")
		if size != "" || imageType != "" {
			var static = imageType == "webp"
			var parsedSize = 0
			if size != "" {
				parsedSize, err = strconv.Atoi(size)
				if err != nil {
					parsedSize = 0
				}
			}
			var url = utils.GenerateImageProxyURL(utils.ImageProxyOptions{URL: finalPath, IsLocalURL: true, Static: static, Size: parsedSize})
			println("Processing Image", url)

			return proxy.Do(c, url)
		}
	}

	println("Serving file:", finalPath)
	return ServeFile(c, finalPath)
}

func ServeFile(c fiber.Ctx, finalPath string) error {
	ext := strings.ToLower(filepath.Ext(finalPath))

	switch {
	case utils.IsOtherMedia(ext):
		// TODO: make it so video doesn't load when directly accessing the url, but only when its embedded in the app.
		return c.SendFile(finalPath, fiber.SendFile{
			ByteRange: true,
			MaxAge:    3600, // 1 Hour
		})
	case utils.IsImage(ext):
		return c.SendFile(finalPath, fiber.SendFile{
			MaxAge: 43200, // 12 Hours
		})
	default:
		return c.Download(finalPath)
	}
}
