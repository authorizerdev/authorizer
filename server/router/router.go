package router

import (
	"yauth/handler"

	"github.com/gofiber/fiber/v2"
)

// SetupRoutes setup router api
func SetupRoutes(app *fiber.App) {
	app.Get("/", handler.Root)
}
