package handler

import "github.com/gofiber/fiber/v2"

// Root hanlde api status
func Root(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "success", "message": "Hello you have reached yauth server", "data": nil})
}
