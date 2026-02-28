package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
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

type ContentHandler struct {
	Env *config.Config
}

func NewContentHandler(context *ContentHandler) *ContentHandler {
	return context
}

func (h *ContentHandler) GetContent(c fiber.Ctx) error {
	var path = c.Path()

	if strings.HasPrefix(path, "/external-embed") {
		var subPath = strings.Join(strings.Split(path, "/")[2:], "/")
		var encryptedPath = strings.Split(subPath, ".")[0]
		decrypted, err := security.Decrypt(encryptedPath, h.Env.ExternalEmbedSecret)
		if err != nil {
			return c.Status(fiber.StatusForbidden).End()
		}
		path = "/" + decrypted
	}

	finalPath, err := resolveSafePath(path)
	if err != nil {
		return c.Status(fiber.StatusForbidden).End()
	}

	// Check file size
	info, err := os.Stat(finalPath)
	if err != nil || info.IsDir() {

		return c.Status(fiber.StatusNotFound).End()
	}

	if shouldProxyImage(c, finalPath, info.Size()) {
		return handleProxyImage(c, finalPath)
	}

	println("Serving file:", finalPath)
	return serveFile(c, finalPath)
}

func serveFile(c fiber.Ctx, finalPath string) error {
	ext := strings.ToLower(filepath.Ext(finalPath))

	switch {
	case utils.IsAudioOrVideo(ext):
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

func resolveSafePath(urlPath string) (string, error) {
	decodedBasename, err := url.PathUnescape(path.Base(urlPath))
	if err != nil {
		return "", err
	}

	// Force path to be relative to prevent escaping to root
	relPath := path.Join(path.Dir(urlPath), decodedBasename)

	relPath = strings.TrimPrefix(relPath, "/")

	baseDir := "./public"
	finalPath := filepath.Join(baseDir, relPath)

	absBase, _ := filepath.Abs(baseDir)
	absFinal, err := filepath.Abs(finalPath)
	if err != nil || !strings.HasPrefix(absFinal, absBase) {
		return "", os.ErrPermission
	}
	return finalPath, nil
}

func shouldProxyImage(c fiber.Ctx, finalPath string, size int64) bool {
	ext := strings.ToLower(filepath.Ext(finalPath))
	fileSizeMB := size / (1024 * 1024) // size in MB
	var isImageFileSize = fileSizeMB <= 20
	var hasTransformParams = c.Query("size") != "" || c.Query("type") != ""

	return utils.IsImage(ext) && isImageFileSize && hasTransformParams

}

func handleProxyImage(c fiber.Ctx, finalPath string) error {
	imageType := c.Query("type")
	size := c.Query("size")
	var static = imageType == "webp"
	var parsedSize = 0
	if size != "" {
		parsedSize, _ = strconv.Atoi(size)
	}
	var proxyURL = utils.GenerateBasicImageProxyURL(utils.BasicImageProxyOptions{URL: finalPath, IsLocalURL: true, Static: static, Size: parsedSize})
	println("Processing Image", proxyURL)

	return proxy.Do(c, proxyURL)
}
