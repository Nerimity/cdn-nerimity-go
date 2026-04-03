package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

type ContentHandler struct {
	Env *config.Config
}

var thumbMutexes sync.Map

func obtainThumbMutex(key string) *sync.Mutex {
	lockInterface, _ := thumbMutexes.LoadOrStore(key, &sync.Mutex{})
	return lockInterface.(*sync.Mutex)
}

func thumbnailCachePath(projectRoot, finalPath string) string {
	cacheDir := filepath.Join(projectRoot, "video-thumb-cache")
	hash := sha256.Sum256([]byte(finalPath))
	filename := hex.EncodeToString(hash[:]) + ".webp"
	return filepath.Join(cacheDir, filename)
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

	return serveFile(c, finalPath)
}

func (h *ContentHandler) GetContentThumb(c fiber.Ctx) error {
	rawPath := c.Path() // "/attachments/xxx/thumb.webp"

	filePath := strings.TrimSuffix(rawPath, "/thumb.webp")
	finalPath, err := resolveSafePath(filePath)
	if err != nil {
		return c.Status(fiber.StatusForbidden).End()
	}

	if !strings.HasPrefix(finalPath, filepath.Clean("./public")) {
		return c.Status(fiber.StatusForbidden).End()
	}

	// Check file size and existence
	info, err := os.Stat(finalPath)
	if err != nil || info.IsDir() {
		return c.Status(fiber.StatusNotFound).End()
	}

	ext := strings.ToLower(filepath.Ext(finalPath))
	if !utils.IsVideo(ext) {
		return c.Status(fiber.StatusBadRequest).SendString("only video thumbnail extraction is supported")
	}

	thumbPath := thumbnailCachePath(h.Env.ProjectRoot, finalPath)
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0o755); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("could not create cache directory")
	}

	lock := obtainThumbMutex(thumbPath)
	lock.Lock()
	defer lock.Unlock()

	if err := deleteStaleThumbnail(thumbPath, 12*time.Hour); err != nil {
		// If we cannot clean stale cache keep going and re-generate.
		fmt.Printf("failed to delete stale thumbnail %q: %v\n", thumbPath, err)
	}

	if _, err := os.Stat(thumbPath); err == nil {
		c.Type("image/webp")
		return serveFile(c, thumbPath)
	}

	filePath, err = utils.GenerateThumbnail(finalPath, thumbPath)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(fmt.Sprintf("thumbnail generation failed: %v", err))
	}

	return serveFile(c, filePath)
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

func deleteStaleThumbnail(path string, maxAge time.Duration) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if time.Since(info.ModTime()) > maxAge {
		return os.Remove(path)
	}

	return nil
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

	return proxy.Do(c, proxyURL)
}
