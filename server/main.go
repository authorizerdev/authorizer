package main

import (
	"log"
	"yauth/router"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	app := fiber.New()
	app.Use(cors.New())

	router.SetupRoutes(app)

	log.Fatal(app.Listen(":3000"))
}
