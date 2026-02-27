package handlers

import (
	"cdn_nerimity_go/config"
	"cdn_nerimity_go/security"
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type InternalHandler struct {
	Env *config.Config
	Jwt *security.JWTService
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
