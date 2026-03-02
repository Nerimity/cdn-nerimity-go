package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type InternalHandler struct {
	Env                *config.Config
	Jwt                *security.JWTService
	PendingFileManager *utils.PendingFilesManager
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
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	body := new(GenerateTokenRequest)

	if err := c.Bind().Body(body); err != nil {
		return err
	}

	parsedId, err := strconv.ParseInt(body.UserId, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user Id")
	}

	token, err := h.Jwt.GenerateToken(parsedId)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to generate token")
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
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	body := new(VerifyFileRequest)

	if err := c.Bind().Body(body); err != nil {
		return err
	}

	userId, err := strconv.ParseInt(body.UserId, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid user Id")
	}

	fileId, err := strconv.ParseInt(body.FileId, 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid file Id")
	}

	groupId := int64(0)
	if body.GroupId != "" {
		groupId, err = strconv.ParseInt(body.GroupId, 10, 64)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid group Id")
		}
	}

	pendingFile, err := h.PendingFileManager.Verify(fileId)
	if pendingFile == nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if pendingFile.UserId != userId {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid UserId")
	}
	if pendingFile.GroupId != 0 && pendingFile.GroupId != groupId {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid GroupId")
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
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid Category type")
	}

	err = os.MkdirAll(filepath.Dir(h.Env.ProjectRoot+"/public/"+newPath), 0755)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create directory.")
	}

	err = os.Rename(h.Env.ProjectRoot+"/"+pendingFile.Path, h.Env.ProjectRoot+"/public/"+newPath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to rename file.")
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
