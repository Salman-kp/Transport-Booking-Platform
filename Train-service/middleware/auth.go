package middleware

import "github.com/gofiber/fiber/v3"

// ExtractUser reads X-User-ID and X-User-Role headers injected by the API Gateway.
// Returns 401 if X-User-ID is missing.
func ExtractUser() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID := c.Get("X-User-ID")
		if userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}
		c.Locals("userID", userID)
		c.Locals("userRole", c.Get("X-User-Role"))
		return c.Next()
	}
}

// RequireRole ensures the authenticated user has a specific role (e.g. "admin").
// Must be used after ExtractUser.
func RequireRole(role string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRole, _ := c.Locals("userRole").(string)
		if userRole != role {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "forbidden: insufficient role",
			})
		}
		return c.Next()
	}
}
