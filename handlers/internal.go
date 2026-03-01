package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"cdn_nerimity_go/utils"
	"fmt"
	"strconv"

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
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("Failed to generate token: %s", err.Error()))
	}

	println(token)

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
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}
	println(groupId, pendingFile.GroupId)
	if pendingFile.GroupId != 0 && pendingFile.GroupId != groupId {
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
	}

	return c.JSON(fiber.Map{
		"file": pendingFile,
	})
}
