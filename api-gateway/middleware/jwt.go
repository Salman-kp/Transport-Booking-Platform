package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/junaid9001/tripneo/api-gateway/config"
)

type Claims struct {
	UserID string
	Role   string
	jwt.RegisteredClaims
}

// Validate JWT and add claims in header for other services
func JwtMiddleware(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		var tokenStr string

		tokenStr = c.Cookies("access_token")

		if tokenStr == "" {
			auth := c.Get("Authorization")

			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}

		}

		if tokenStr == "" {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
			}
			return []byte(cfg.JWT_SECRET), nil
		})

		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}

		if !token.Valid {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}

		c.Request().Header.Set("X-User-ID", claims.UserID)
		c.Request().Header.Set("X-User-Role", claims.Role)

		//for ratelimit
		c.Locals("userID", claims.UserID)

		return c.Next()
	}
}
