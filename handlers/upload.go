package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	Env   *config.Config
	Flake *utils.Flake
	Jwt   *security.JWTService
}

func NewUploadHandler(context *UploadHandler) *UploadHandler {
	return context
}

const MaxUploadSize = 20 * 1024 * 1024

func (h *UploadHandler) UploadFile(c fiber.Ctx) error {
	contentLength := c.Request().Header.ContentLength()
	filename := c.Get("File-Name")
	token := c.Get("Authorization")

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	claims, err := h.Jwt.VerifyToken(token)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}
	println(claims.UserID)

	safeFilename := utils.SafeFilename(filename)
	ext := filepath.Ext(safeFilename)

	if contentLength > MaxUploadSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("File too large")
	}
	if contentLength <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid content length")
	}

	id := h.Flake.Generate()

	filePath := "temp/" + strconv.FormatInt(id, 10) + ext
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	uploadSuccessful := false

	defer func() {
		println("done")
		file.Close()
		if !uploadSuccessful {
			os.Remove(filePath)
		}
	}()

	src := c.Request().BodyStream()

	buf := make([]byte, 1024*1024)
	limitSrc := io.LimitReader(src, MaxUploadSize+1)
	written, err := io.CopyBuffer(struct{ io.Writer }{file}, limitSrc, buf)
	if err != nil {
		return err
	}

	if written > MaxUploadSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("File exceeds size limit")
	}

	uploadSuccessful = true
	return c.SendString("Uploaded!")

}
