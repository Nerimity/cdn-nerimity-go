package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cshum/vipsgen/vips"
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
	mimeType := c.Get("File-Content-Type")
	token := c.Get("Authorization")

	groupId := c.Params("groupId")

	attachmentCategory := utils.FileCategory(strings.ToLower(strings.Split(c.Path(), "/")[1]))
	isImage := utils.IsImage(filepath.Ext(filename))

	println(attachmentCategory, groupId)

	if token == "" {
		return sendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	claims, err := h.Jwt.VerifyToken(token)

	if err != nil {
		return sendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	safeFilename := utils.SafeFilename(filename)
	ext := filepath.Ext(safeFilename)

	if contentLength > MaxUploadSize {
		return sendError(c, fiber.StatusBadRequest, "File too large")
	}
	if contentLength <= 0 {
		return sendError(c, fiber.StatusBadRequest, "Invalid content length")
	}

	fileId := h.Flake.Generate()

	filePath := "temp/" + strconv.FormatInt(fileId, 10) + ext
	file, err := os.Create(filePath)
	if err != nil {
		return sendError(c, fiber.StatusInternalServerError, "Failed to create file")
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
		return sendError(c, fiber.StatusInternalServerError, "Failed to write file")
	}

	if written > MaxUploadSize {
		return sendError(c, fiber.StatusBadRequest, "File exceeds size limit")
	}

	shouldCompressImage := isImage && written <= MaxImageSize
	if attachmentCategory != "attachments" {
		if !isImage {
			return sendError(c, fiber.StatusBadRequest, "Invalid file type")
		}
		if !shouldCompressImage {
			return sendError(c, fiber.StatusBadRequest, "Image exceeds size limit")
		}
	}

	imageCompressed := false
	if shouldCompressImage {
		newPath, err := compressImage(c, filePath, attachmentCategory)
		if err != nil && attachmentCategory != utils.AttachmentsCategory {
			return sendError(c, fiber.StatusInternalServerError, "Failed to compress image")
		}

		if err == nil {
			imageCompressed = true
			filePath = newPath
		}
	}

	pendingFile := utils.PendingFile{
		FileId:          fileId,
		Path:            filePath,
		Type:            attachmentCategory,
		ImageCompressed: imageCompressed,
		MimeType:        mimeType,
		FileSize:        int(written),
		ExpiresAt:       time.Now().Add(2 * time.Minute),
	}

	if imageCompressed {
		image, err := vips.NewImageFromFile(filePath, nil)
		if err != nil {
			return sendError(c, fiber.StatusInternalServerError, "Failed to open image")
		}
		defer image.Close()
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return sendError(c, fiber.StatusInternalServerError, "Failed to get file info")
		}
		pendingFile.Height = image.Height()
		pendingFile.Width = image.Width()
		pendingFile.Animated = image.Pages() > 1
		pendingFile.FileSize = int(fileInfo.Size())
	}
	if groupId != "" {
		pendingFile.GroupId, err = strconv.ParseInt(groupId, 10, 64)
		if err != nil {
			return sendError(c, fiber.StatusBadRequest, "Invalid group id")
		}
	}
	if claims.UserId != "" {
		pendingFile.UserId, _ = strconv.ParseInt(claims.UserId, 10, 64)
	}

	h.PendingFilesManager.Add(pendingFile)

	uploadSuccessful = true
	return c.JSON(fiber.Map{
		"fileId": strconv.FormatInt(fileId, 10),
	})

}

func sendError(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"message": message,
	})
}

func compressImage(c fiber.Ctx, filePath string, category utils.FileCategory) (string, error) {
	opts := utils.ImageProxyOptions{
		Path: filePath,
	}
	opts.Size = getSize(category)

	if category == utils.AvatarsCategory || category == utils.ProfileBannersCategory {
		strPoints := c.Query("points")
		dimensions, points, _ := utils.PointsToDimensions(strPoints)
		if dimensions != nil {
			opts.Crop = &utils.ImageProxyCrop{
				Width:  dimensions.Width,
				Height: dimensions.Height,
				X:      int(math.Round(points[0])),
				Y:      int(math.Round(points[1])),
			}
		}

	}

	url, err := utils.GenerateImageProxyURL(opts)
	if err != nil {
		return "", err
	}
	newPath, err := downloadAndReplaceImage(url, filePath)
	if err != nil {
		return "", err
	}

	return newPath, nil

}

func getSize(category utils.FileCategory) utils.ImageProxySize {
	size := utils.ImageProxySize{}

	if category == utils.AttachmentsCategory {
		size.Width = 1920
		size.Height = 1080
		size.ResizeType = utils.ResizeTypeFit
	}
	if category == utils.EmojisCategory {
		size.Width = 100
		size.Height = 100
		size.ResizeType = utils.ResizeTypeFit
	}
	if category == utils.AvatarsCategory {
		size.Width = 200
		size.Height = 200
		size.ResizeType = utils.ResizeTypeFill
	}
	if category == utils.ProfileBannersCategory {
		size.Width = 1920
		size.Height = 1080
		size.ResizeType = utils.ResizeTypeFill
	}
	return size
}

func downloadAndReplaceImage(url, oldFilePath string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch: %s", resp.Status)
	}

	dir := filepath.Dir(oldFilePath)
	base := filepath.Base(oldFilePath)
	nameWithoutExt := strings.TrimSuffix(base, filepath.Ext(base))
	newPath := filepath.Join(dir, nameWithoutExt+".webp")

	tempFile, err := os.CreateTemp(dir, "replace-*.tmp")
	if err != nil {
		return "", err
	}
	tempName := tempFile.Name()

	defer func() {
		tempFile.Close()
		os.Remove(tempName)
	}()

	buffer := make([]byte, 1024*1024)

	_, err = io.CopyBuffer(tempFile, resp.Body, buffer)
	if err != nil {
		return "", err
	}

	if err := tempFile.Close(); err != nil {
		return "", err
	}

	if oldFilePath != newPath {
		_ = os.Remove(oldFilePath)
	}

	err = os.Rename(tempName, newPath)
	if err != nil {
		return "", err
	}

	return newPath, nil
}
