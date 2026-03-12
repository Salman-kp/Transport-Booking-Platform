package proxy

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/proxy"
)

// reverse proxy
func To(baseURL string) fiber.Handler {
	return func(c fiber.Ctx) error {
		target := baseURL + c.OriginalURL()
		log.Println(target)
		log.Println(c.OriginalURL())

		if err := proxy.Do(c, target); err != nil {
			return c.Status(502).JSON(fiber.Map{"error": "service unavailable"})
		}

		return nil
	}
}
