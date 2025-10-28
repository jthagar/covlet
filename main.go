package main

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

func main() {
	app := fiber.New()
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		test := []string{}
		test = append(test, "home")
		test = append(test, "about")
		encoder := json.NewEncoder
		return encoder(c).Encode(test)
	})

	app.Get("/home", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Hello, World!",
		})
	})

	err := app.Listen(":3000")
	if err != nil {
		return
	}
}
