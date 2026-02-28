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
	filename := c.Get("File-Name")
	groupId := c.Params("groupId")
	fileContentType := string(c.Request().Header.ContentType())

	isImage := utils.IsImage(filepath.Ext(filename)) && utils.IsMimeImage(fileContentType)

	claims, err := auth(c, h)
	if err != nil {
		return err
	}

	err = validate(c, h)
	if err != nil {
		return err
	}
	var finalPath string
	success := false
	defer func() {
		if !success && finalPath != "" {
			os.Remove(finalPath)
		}
	}()

	pendingFile, err := handleUpload(c, h)
	if err != nil {
		return err
	}
	finalPath = pendingFile.Path

	shouldCompressImage := isImage && pendingFile.FileSize <= MaxImageSize

	imageCompressed := false
	if shouldCompressImage {
		imageCompressed, err = handleCompressImage(c, h, pendingFile)
		if err != nil {
			return err
		}
	}

	if imageCompressed {
		finalPath = pendingFile.Path
		err = handleImageMetadata(c, pendingFile)
		if err != nil {
			return err
		}

	}
	if groupId != "" {
		pendingFile.GroupId, _ = strconv.ParseInt(groupId, 10, 64)

	}
	if claims.UserId != "" {
		pendingFile.UserId, _ = strconv.ParseInt(claims.UserId, 10, 64)
	}

	pendingFile.ExpiresAt = time.Now().Add(1 * time.Minute)
	h.PendingFilesManager.Add(*pendingFile)
	success = true
	return c.JSON(fiber.Map{
		"fileId": strconv.FormatInt(pendingFile.FileId, 10),
	})

}
func handleImageMetadata(c fiber.Ctx, pendingFile *utils.PendingFile) error {
	pendingFile.ImageCompressed = true
	image, err := vips.NewImageFromFile(pendingFile.Path, nil)
	if err != nil {
		return sendError(c, fiber.StatusInternalServerError, "Failed to open image")
	}
	defer image.Close()
	fileInfo, err := os.Stat(pendingFile.Path)
	if err != nil {
		return sendError(c, fiber.StatusInternalServerError, "Failed to get file info")
	}
	pendingFile.Height = image.Height()
	pendingFile.Width = image.Width()
	pendingFile.Animated = image.Pages() > 1
	pendingFile.FileSize = int(fileInfo.Size())
	pendingFile.MimeType = "image/webp"

	return nil
}

func sendError(c fiber.Ctx, status int, message string) error {
	c.Status(status).JSON(fiber.Map{
		"message": message,
	})
	return fiber.NewError(status, message)
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

func auth(c fiber.Ctx, h *UploadHandler) (*security.Claims, error) {
	token := c.Get("Authorization")
	if token == "" {
		return nil, sendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	claims, err := h.Jwt.VerifyToken(token)

	if err != nil {
		return nil, sendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	return claims, nil

}

func validate(c fiber.Ctx, h *UploadHandler) error {
	contentLength := c.Request().Header.ContentLength()
	filename := c.Get("File-Name")
	fileContentType := c.Request().Header.ContentType()

	groupId := c.Params("groupId")

	attachmentCategory := utils.FileCategory(strings.ToLower(strings.Split(c.Path(), "/")[1]))
	isImage := utils.IsImage(filepath.Ext(filename)) && utils.IsMimeImage(fileContentType)

	if contentLength > MaxUploadSize {
		return sendError(c, fiber.StatusBadRequest, "File too large")
	}
	if contentLength <= 0 {
		return sendError(c, fiber.StatusBadRequest, "Invalid content length")
	}

	shouldCompressImage := isImage && contentLength <= MaxImageSize
	if attachmentCategory != "attachments" {
		if !isImage {
			return sendError(c, fiber.StatusBadRequest, "Invalid file type")
		}
		if !shouldCompressImage {
			return sendError(c, fiber.StatusBadRequest, "Image exceeds size limit")
		}
	}

	if groupId != "" {
		_, err := strconv.ParseInt(groupId, 10, 64)
		if err != nil {
			return sendError(c, fiber.StatusBadRequest, "Invalid group id")
		}
	}

	return nil
}

func handleUpload(c fiber.Ctx, h *UploadHandler) (*utils.PendingFile, error) {
	filename := c.Get("File-Name")
	mimeType := string(c.Request().Header.ContentType())

	attachmentCategory := utils.FileCategory(strings.ToLower(strings.Split(c.Path(), "/")[1]))

	safeFilename := utils.SafeFilename(filename)
	ext := filepath.Ext(safeFilename)

	fileId := h.Flake.Generate()

	filePath := "temp/" + strconv.FormatInt(fileId, 10) + ext
	file, err := os.Create(filePath)
	if err != nil {
		return nil, sendError(c, fiber.StatusInternalServerError, "Failed to create file")
	}
	defer file.Close()

	src := c.Request().BodyStream()

	buf := make([]byte, 1024*1024)
	limitSrc := io.LimitReader(src, MaxUploadSize+1)
	written, err := io.CopyBuffer(struct{ io.Writer }{file}, limitSrc, buf)
	if err != nil {
		return nil, sendError(c, fiber.StatusInternalServerError, "Failed to write file")
	}

	if written > MaxUploadSize {
		return nil, sendError(c, fiber.StatusBadRequest, "File exceeds size limit")
	}

	return &utils.PendingFile{
		FileId:   fileId,
		Path:     filePath,
		Type:     attachmentCategory,
		MimeType: mimeType,
		FileSize: int(written),
	}, nil
}

func handleCompressImage(c fiber.Ctx, h *UploadHandler, pendingFile *utils.PendingFile) (bool, error) {
	newPath, err := compressImage(c, pendingFile.Path, pendingFile.Type)
	if err != nil && pendingFile.Type != utils.AttachmentsCategory {
		return false, sendError(c, fiber.StatusInternalServerError, "Failed to compress image")
	}

	if err == nil {
		pendingFile.Path = newPath
		return true, nil
	}

	return false, nil
}
