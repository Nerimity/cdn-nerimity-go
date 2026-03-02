package handlers

import (
	"cdn_nerimity_go/config"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cshum/vipsgen/vips"
	"github.com/gofiber/fiber/v3"
)

type ProxyHandler struct {
	Env *config.Config
}

func NewProxyHandler(context *ProxyHandler) *ProxyHandler {
	return context
}

const (
	maxImageSize   = 12 << 20 // 12MB
	requestTimeout = 20 * time.Second
)

func (h *ProxyHandler) GetImageDimensions(c fiber.Ctx) error {
	rawURL := c.Query("url")
	if rawURL == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing url parameter")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || !isValidScheme(parsed.Scheme) {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid URL")
	}

	if err := validateHost(parsed.Hostname()); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Blocked host")
	}

	client := &http.Client{
		Timeout: requestTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			return validateHost(req.URL.Hostname())
		},
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, rawURL, nil)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Request creation failed")
	}

	resp, err := client.Do(req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("Fetch failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.Status(fiber.StatusBadGateway).SendString("Remote server error")
	}

	if resp.ContentLength > maxImageSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("Image too large")
	}

	tmpFile, err := os.CreateTemp("", "img-*")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Temp file error")
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	written, err := io.Copy(tmpFile, io.LimitReader(resp.Body, maxImageSize))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Download error")
	}

	if written >= maxImageSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("Image too large")
	}

	img, err := vips.NewImageFromFile(tmpFile.Name(), nil)
	if err != nil {
		log.Printf("Vips error: %v", err)
		return c.Status(fiber.StatusUnsupportedMediaType).SendString("Invalid image")
	}
	defer img.Close()

	return c.JSON(fiber.Map{
		"width":    img.Width(),
		"height":   img.Height(),
		"animated": img.Pages() > 1,
	})
}

func isValidScheme(scheme string) bool {
	return scheme == "http" || scheme == "https"
}

func validateHost(host string) error {
	ips, err := net.LookupIP(host)
	if err != nil {
		return err
	}

	for _, ip := range ips {
		if isPrivateIP(ip) {
			return errors.New("private IP not allowed")
		}
	}
	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() {
		return true
	}

	if ip.String() == "::1" {
		return true
	}

	return false
}

// func shouldProxyImage(c fiber.Ctx, finalPath string, size int64) bool {
// 	ext := strings.ToLower(filepath.Ext(finalPath))
// 	fileSizeMB := size / (1024 * 1024) // size in MB
// 	var isImageFileSize = fileSizeMB <= 20
// 	var hasTransformParams = c.Query("size") != "" || c.Query("type") != ""

// 	return utils.IsImage(ext) && isImageFileSize && hasTransformParams

// }

// func handleProxyImage(c fiber.Ctx, finalPath string) error {
// 	imageType := c.Query("type")
// 	size := c.Query("size")
// 	var static = imageType == "webp"
// 	var parsedSize = 0
// 	if size != "" {
// 		parsedSize, _ = strconv.Atoi(size)
// 	}
// 	var proxyURL = utils.GenerateBasicImageProxyURL(utils.BasicImageProxyOptions{URL: finalPath, IsLocalURL: true, Static: static, Size: parsedSize})

// 	return proxy.Do(c, proxyURL)
// }

func isUrl(url string) bool {
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		return true
	}

	return false
}
