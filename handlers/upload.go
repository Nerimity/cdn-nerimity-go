package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	Env                 *config.Config
	Flake               *utils.Flake
	Jwt                 *security.JWTService
	PendingFilesManager *utils.PendingFilesManager
}

func NewUploadHandler(context *UploadHandler) *UploadHandler {
	return context
}

const MaxUploadSize = 50 * 1024 * 1024
const MaxImageSize = 12 * 1024 * 1024

func (h *UploadHandler) UploadFile(c fiber.Ctx) error {
	contentLength := c.Request().Header.ContentLength()
	filename := c.Get("File-Name")
	token := c.Get("Authorization")

	groupId := c.Params("groupId")

	// attachments, emojis, avatars, profile_banners
	attachmentType := strings.ToLower(strings.Split(c.Path(), "/")[1])
	isImage := utils.IsImage(filepath.Ext(filename))

	println(attachmentType, groupId)

	if token == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	claims, err := h.Jwt.VerifyToken(token)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}
	println(claims.UserId)

	safeFilename := utils.SafeFilename(filename)
	ext := filepath.Ext(safeFilename)

	if contentLength > MaxUploadSize {
		return c.Status(fiber.StatusRequestEntityTooLarge).SendString("File too large")
	}
	if contentLength <= 0 {
		return c.Status(fiber.StatusBadRequest).SendString("Invalid content length")
	}

	fileId := h.Flake.Generate()

	filePath := "temp/" + strconv.FormatInt(fileId, 10) + ext
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

	shouldCompressImage := isImage && written <= MaxImageSize
	if attachmentType != "attachments" {
		if !isImage {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid file type")
		}
		if !shouldCompressImage {
			return c.Status(fiber.StatusRequestEntityTooLarge).SendString("Image exceeds size limit")
		}
	}

	if shouldCompressImage {
		utils.GenerateImageProxyURL(utils.ImageProxyOptions{
			Path: filePath,
		})
	}

	pendingFile := utils.PendingFile{
		FileId:    fileId,
		Path:      filePath,
		Type:      attachmentType,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	if groupId != "" {
		pendingFile.GroupId, _ = strconv.ParseInt(groupId, 10, 64)
	}
	if claims.UserId != "" {
		pendingFile.UserId, _ = strconv.ParseInt(claims.UserId, 10, 64)
	}

	h.PendingFilesManager.Add(pendingFile)

	uploadSuccessful = true
	return c.SendString("Uploaded!")

}
