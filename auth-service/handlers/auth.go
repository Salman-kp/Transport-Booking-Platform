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
		token, user, err := service.Login(cfg, req.Email, req.Password)
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
			"user": fiber.Map{
				"email": user.Email,
				"role":  user.Role,
			},
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

func AssignRole() fiber.Handler {
	return func(c fiber.Ctx) error {
		role := c.Get("X-User-Role")
		if role != "superadmin" {
			return c.Status(403).JSON(fiber.Map{"error": "forbidden: superadmin only"})
		}

		var req struct {
			Email       string   `json:"email" validate:"required,email"`
			Role        string   `json:"role" validate:"required"`
			Permissions []string `json:"permissions" validate:"required"`
		}

		if err := c.Bind().Body(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid JSON body"})
		}

		if err := Validate.Struct(req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid request body: missing fields"})
		}

		err := service.AssignRole(req.Email, req.Role, req.Permissions)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		return c.Status(200).JSON(fiber.Map{"message": "role and permissions assigned successfully"})
	}
}

func ListUsers() fiber.Handler {
	return func(c fiber.Ctx) error {
		role := c.Get("X-User-Role")
		if role != "superadmin" && role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "forbidden: admin only"})
		}

		users, err := service.ListUsers()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		type UserDTO struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Email      string `json:"email"`
			Role       string `json:"role"`
			IsVerified bool   `json:"is_verified"`
		}
		dtos := make([]UserDTO, 0, len(users))
		for _, u := range users {
			dtos = append(dtos, UserDTO{
				ID:         u.ID.String(),
				Name:       u.Name,
				Email:      u.Email,
				Role:       u.Role,
				IsVerified: u.IsVerified,
			})
		}

		return c.Status(200).JSON(fiber.Map{"users": dtos})
	}
}

// Internal endpoint for service-to-service communication
func GetUserByID() fiber.Handler {
	return func(c fiber.Ctx) error {
		id := c.Params("id")
		if id == "" {
			return c.Status(400).JSON(fiber.Map{"error": "user id is required"})
		}

		user, err := service.GetUserByID(id)
		if err != nil {
			return c.Status(404).JSON(fiber.Map{"error": "user not found"})
		}

		return c.Status(200).JSON(fiber.Map{
			"id":    user.ID.String(),
			"name":  user.Name,
			"email": user.Email,
		})
	}
}
