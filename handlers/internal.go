package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/database"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
)

type InternalHandler struct {
	Env                *config.Config
	Jwt                *security.JWTService
	PendingFileManager *utils.PendingFilesManager
	Database           *database.DatabaseService
}

func NewInternalHandler(context *InternalHandler) *InternalHandler {
	return context
}

type GenerateTokenRequest struct {
	UserId string `json:"userId"`
}

func (h *InternalHandler) GenerateToken(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader != h.Env.InternalSecret {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	body := new(GenerateTokenRequest)

	if err := c.Bind().Body(body); err != nil {
		return err
	}

	parsedId, err := strconv.ParseInt(body.UserId, 10, 64)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid user Id")
	}

	token, err := h.Jwt.GenerateToken(parsedId)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to generate token")
	}

	return c.JSON(fiber.Map{
		"token": token,
	})
}

type VerifyFileRequest struct {
	UserId  string `json:"userId"`
	FileId  string `json:"fileId"`
	GroupId string `json:"groupId"`
}

func (h *InternalHandler) VerifyFile(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader != h.Env.InternalSecret {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}
	body := new(VerifyFileRequest)

	if err := c.Bind().Body(body); err != nil {
		return err
	}

	userId, err := strconv.ParseInt(body.UserId, 10, 64)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid user Id")
	}

	fileId, err := strconv.ParseInt(body.FileId, 10, 64)
	if err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid file Id")
	}

	groupId := int64(0)
	if body.GroupId != "" {
		groupId, err = strconv.ParseInt(body.GroupId, 10, 64)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, "Invalid group Id")
		}
	}

	pendingFile, err := h.PendingFileManager.Verify(fileId)
	if pendingFile == nil {
		return utils.SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if pendingFile.UserId != userId {
		return utils.SendError(c, fiber.StatusUnauthorized, "Invalid UserId")
	}
	if pendingFile.GroupId != 0 && pendingFile.GroupId != groupId {
		return utils.SendError(c, fiber.StatusUnauthorized, "Invalid GroupId")
	}

	var newPath = ""
	if pendingFile.Type == utils.ProfileBannersCategory || pendingFile.Type == utils.AvatarsCategory {
		if pendingFile.Type == utils.ProfileBannersCategory {
			newPath = "profile_banners/"
		} else {
			newPath = "avatars/"
		}

		newPath += strconv.FormatInt(groupId, 10) + "/"
		newPath += pendingFile.Filename
	}
	if pendingFile.Type == utils.EmojisCategory {
		newPath = "emojis/" + pendingFile.Filename
	}
	if pendingFile.Type == utils.AttachmentsCategory {
		origName := filepath.Base(pendingFile.OriginalFilename)
		origExt := filepath.Ext(pendingFile.OriginalFilename)
		origNameWithoutExt := strings.TrimSuffix(origName, origExt)
		newPath = "attachments/" + strconv.FormatInt(groupId, 10) + "/" + strconv.FormatInt(fileId, 10) + "/" + origNameWithoutExt + filepath.Ext(pendingFile.Filename)
	}

	if newPath == "" {
		return utils.SendError(c, fiber.StatusUnauthorized, "Invalid Category type")
	}

	var expireAt int64
	if !pendingFile.ImageCompressed {
		createdAt, err := h.Database.AddExpire(fileId, groupId)
		if err != nil {
			log.Println(err)
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to add expire.")
		}

		futureTime := createdAt.Add(12 * time.Hour)
		expireAt = futureTime.UnixMilli()

	}

	err = os.MkdirAll(filepath.Dir(h.Env.ProjectRoot+"/public/"+newPath), 0755)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create directory.")
	}

	err = os.Rename(h.Env.ProjectRoot+"/"+pendingFile.Path, h.Env.ProjectRoot+"/public/"+newPath)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to rename file.")
	}

	name := filepath.Base(newPath)
	Ext := filepath.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, Ext)

	path := filepath.Dir(newPath) + "/" + utils.EncodeURIComponent(nameWithoutExt) + Ext

	if pendingFile.Animated {
		path += "#a"
	}

	json := fiber.Map{
		"fileId":     strconv.FormatInt(pendingFile.FileId, 10),
		"path":       strings.ReplaceAll(path, "\\", "/"),
		"filesize":   pendingFile.FileSize,
		"animated":   pendingFile.Animated,
		"mimetype":   pendingFile.MimeType,
		"compressed": pendingFile.ImageCompressed,
	}
	if expireAt > 0 {
		json["expireAt"] = expireAt
	}
	if pendingFile.Duration > 0 {
		json["duration"] = pendingFile.Duration
	}
	if pendingFile.Height > 0 {
		json["height"] = pendingFile.Height
	}
	if pendingFile.Width > 0 {
		json["width"] = pendingFile.Width
	}

	return c.JSON(json)
}

func (h *InternalHandler) DeleteByFileIds(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader != h.Env.InternalSecret {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var body struct {
		Paths []string `json:"paths"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	if body.Paths == nil {
		return c.Status(400).JSON(fiber.Map{"error": "Missing paths"})
	}

	for _, path := range body.Paths {
		if path == "" {
			continue
		}

		if strings.HasSuffix(path, "#a") {
			path = strings.TrimSuffix(path, "#a")
		}
		utils.DeleteRecursiveEmpty(h.Env.ProjectRoot + "/public/" + path)
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
		"count":  len(body.Paths),
	})
}

func (h *InternalHandler) DeleteAttachmentsByGroupId(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader != h.Env.InternalSecret {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	groupId := c.Params("groupId")
	DELETE_BATCH := 1000
	groupPath := h.Env.ProjectRoot + "/public/attachments/" + groupId

	f, err := os.Open(groupPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid Path", "type": "INVALID_PATH"})
	}
	defer f.Close()

	entries, err := f.ReadDir(DELETE_BATCH)

	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Error during iteration.")
	}

	for _, entry := range entries {
		fullPath := filepath.Join(groupPath, entry.Name())
		os.RemoveAll(fullPath)
	}

	os.Remove(groupPath)

	return c.JSON(fiber.Map{
		"status": "deleted",
		"count":  len(entries),
	})
}

func (h *InternalHandler) DeleteFile(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")

	if authHeader != h.Env.InternalSecret {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized")
	}

	var body struct {
		Path string `json:"path"`
	}

	if err := c.Bind().Body(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
	}

	if body.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Missing path"})
	}

	if strings.HasSuffix(body.Path, "#a") {
		body.Path = strings.TrimSuffix(body.Path, "#a")
	}

	decodedPath, err := utils.DecodeURIComponent(body.Path)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid path"})
	}
	fullPath := h.Env.ProjectRoot + "/public/" + decodedPath

	err = utils.DeleteRecursiveEmpty(fullPath)

	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Failed to delete file"})
	}

	return c.JSON(fiber.Map{
		"status": "deleted",
	})
}
