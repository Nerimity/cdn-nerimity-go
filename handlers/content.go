package handlers

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
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
		return c.Status(fiber.StatusForbidden).SendString("Access denied")
	}

	ext := filepath.Ext(finalPath)
	// Check file size
	info, err := os.Stat(finalPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("File not found")
	}
	fileSizeMB := info.Size() / (1024 * 1024) // size in MB
	var isImageFileSize = fileSizeMB <= 20

	if isImage(strings.ToLower(ext)) && isImageFileSize {
		imageType := c.Query("type")
		size := c.Query("size")
		var static = imageType == "webp"
		var parsedSize = 0
		if size != "" {
			parsedSize, err = strconv.Atoi(size)
			if err != nil {
				parsedSize = 0
			}
		}
		println("Processing Image", GenerateImageProxyURL(ImageProxyOptions{URL: finalPath, IsLocalURL: true, Static: static, Size: parsedSize}))
	}

	println("Serving file:", finalPath)
	return ServeFile(c, finalPath)
}

func ServeFile(c fiber.Ctx, finalPath string) error {
	ext := strings.ToLower(filepath.Ext(finalPath))

	switch {
	case isOtherMedia(ext):
		// TODO: make it so video doesn't load when directly accessing the url, but only when its embedded in the app.
		return c.SendFile(finalPath, fiber.SendFile{
			ByteRange: true,
			MaxAge:    3600, // 1 Hour
		})
	case isImage(ext):
		return c.SendFile(finalPath, fiber.SendFile{
			MaxAge: 43200, // 12 Hours
		})
	default:
		return c.Download(finalPath)
	}
}

func isOtherMedia(ext string) bool {
	switch ext {
	case ".mp4", ".webm", ".ogg", ".mp3", ".wav":
		return true
	default:
		return false
	}
}

func isImage(ext string) bool {
	switch ext {
	case ".webp", ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}

type ImageProxyOptions struct {
	URL        string
	IsLocalURL bool
	Static     bool
	Size       int
}

const BASE_PROXY = "http://localhost:8888/pr:sharp/"

func GenerateImageProxyURL(opts ImageProxyOptions) string {
	var parts []string

	var path = opts.URL
	if opts.IsLocalURL {
		path = "local:///" + path
	}
	var encodedPath = encodeURIComponent(path)

	if opts.Static {
		var static = "page:0"
		parts = append(parts, static)
	}

	if opts.Size != 0 {
		var size = fmt.Sprintf("rs:fit:%d:%d", opts.Size, opts.Size)
		parts = append(parts, size)
	}

	parts = append(parts, encodedPath)

	return BASE_PROXY + strings.Join(parts, "/") + "@webp"

}

func encodeURIComponent(str string) string {
	escaped := url.QueryEscape(str)
	// replace + with %20 to match JavaScript's encodeURIComponent
	return strings.ReplaceAll(escaped, "+", "%20")
}
