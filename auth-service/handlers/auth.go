package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/junaid9001/tripneo/auth-service/config"
	domainerrors "github.com/junaid9001/tripneo/auth-service/domain_errors"
	"github.com/junaid9001/tripneo/auth-service/service"
	"github.com/redis/go-redis/v9"
)

var Validate = validator.New()

func Register(rdb *redis.Client, cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Name     string `json:"name" validate:"required"`
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=6"`
		}

		if err := c.Bind().Body(&req); err != nil {
			log.Print(err)
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		if err := Validate.Struct(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		err := service.CreateUser(c.RequestCtx(), cfg, rdb, req.Name, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domainerrors.EmailAlreadyTaken) {
				return c.Status(http.StatusConflict).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(http.StatusCreated).JSON(fiber.Map{
			"message": "user registered successfully, please verify your email",
		})

	}

}

func VerifyOtp(rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Email string `json:"email" validate:"required,email"`
			Otp   string `json:"otp" validate:"required,min=6"`
		}

		if err := c.Bind().Body(&req); err != nil {
			log.Print(err)
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		if err := Validate.Struct(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		err := service.ValidateOtp(c.RequestCtx(), rdb, req.Email, req.Otp)
		if err != nil {
			if errors.Is(err, domainerrors.EmailNotFound) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.ErrInvalidOrExpiredOtp) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.EmailALreadyVerified) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}

			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"message": "email verified successfully",
		})

	}
}

func ResendOtp(cfg *config.Config, rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Email string `json:"email" validate:"required,email"`
		}

		if err := c.Bind().Body(&req); err != nil {
			log.Print(err)
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		if err := Validate.Struct(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		err := service.ResendOtp(c.RequestCtx(), cfg, rdb, req.Email)
		if err != nil {
			if errors.Is(err, domainerrors.EmailNotFound) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.ResendOtpCooldown) {

				return c.Status(429).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.EmailALreadyVerified) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			}
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})

		}

		return c.Status(http.StatusOK).JSON(fiber.Map{"message": "new otp sended to your email"})
	}

}
func Login(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		var req struct {
			Email    string `json:"email" validate:"required,email"`
			Password string `json:"password" validate:"required,min=6"`
		}

		if err := c.Bind().Body(&req); err != nil {
			log.Print(err)
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}

		if err := Validate.Struct(req); err != nil {
			return c.Status(http.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid JSON body",
			})
		}
		token, err := service.Login(cfg, req.Email, req.Password)
		if err != nil {
			if errors.Is(err, domainerrors.EmailNotFound) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.InvalidEmailOrPassword) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})

			} else if errors.Is(err, domainerrors.VerifyEmailBeforeLoggingIN) {

				return c.Status(400).JSON(fiber.Map{"error": err.Error()})
			} else {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}
		}

		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    token,
			HTTPOnly: true,
			Secure:   false, //true in prod
			SameSite: "Lax",
			Path:     "/",
			MaxAge:   60 * 60 * 24,
		})

		return c.Status(http.StatusOK).JSON(fiber.Map{
			"message": "login successful",
			"token":   token,
		})

	}
}

func Logout() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Cookie(&fiber.Cookie{
			Name:     "access_token",
			Value:    "",
			HTTPOnly: true,
			Secure:   false, //true in prod
			SameSite: "Lax",
			Path:     "/",
			MaxAge:   -1,
		})
		return c.Status(200).JSON(fiber.Map{"message": "logout successful"})
	}
}
