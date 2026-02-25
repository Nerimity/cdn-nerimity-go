package handlers

import (
	"cdn_nerimity_go/config"
	"io"
	"os"

	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	Env *config.Config
}

func NewUploadHandler(env *config.Config) *UploadHandler {
	return &UploadHandler{Env: env}
}

const MaxUploadSize = 20 * 1024 * 1024

func (h *UploadHandler) UploadFile(c fiber.Ctx) error {
	contentLength := c.Request().Header.ContentLength()

	// if contentLength > MaxUploadSize {
	// 	return c.Status(fiber.StatusRequestEntityTooLarge).SendString("File too large")
	// }
	if contentLength <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid content length")
	}

	filePath := "uploaded_file"
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
	_, err = io.CopyBuffer(struct{ io.Writer }{file}, limitSrc, buf)
	if err != nil {
		return err
	}

	info, _ := file.Stat()
	written := info.Size()

	if written > MaxUploadSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("File exceeds size limit")
	}

	uploadSuccessful = true
	return c.SendString("Uploaded!")

}
