package server

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/jthagar/covlet/backend/pkg/config"
)

// New creates the Fiber application with routes and middleware.
func New() *fiber.App {
	config.InitMainDir()
	_, _ = config.EnsureTemplatesDir()

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		BodyLimit:             512 * 1024, // template uploads (PUT /file)
	})
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type",
	}))

	registerAPI(app)

	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "covlet",
			"api":     "/api/v1",
		})
	})

	return app
}

// Listen starts the HTTP server on addr (e.g. ":8080").
func Listen(addr string) error {
	if addr == "" {
		addr = ":8080"
	}
	return New().Listen(addr)
}

// AddrFromEnv returns COVLET_LISTEN or ":8080".
func AddrFromEnv() string {
	if a := os.Getenv("COVLET_LISTEN"); a != "" {
		return a
	}
	if p := os.Getenv("COVLET_PORT"); p != "" {
		if p[0] == ':' {
			return p
		}
		return ":" + p
	}
	return ":8080"
}
