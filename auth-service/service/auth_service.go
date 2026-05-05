package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/junaid9001/tripneo/auth-service/config"
	domainerrors "github.com/junaid9001/tripneo/auth-service/domain_errors"
	"github.com/junaid9001/tripneo/auth-service/models"
	"github.com/junaid9001/tripneo/auth-service/repository"
	"github.com/junaid9001/tripneo/auth-service/utils"
	"github.com/redis/go-redis/v9"
)

func CreateUser(ctx context.Context, cfg *config.Config, rdb *redis.Client, name, email, password string) error {
	hashedPass, err := utils.GenerateHashedPassword(password)
	if err != nil {
		return errors.New("internal server error")
	}
	err = repository.InsertUser(name, email, hashedPass)
	if err != nil {
		if errors.Is(err, domainerrors.EmailAlreadyTaken) {
			return domainerrors.EmailAlreadyTaken
		}
		return errors.New("internal server error")
	}

	otp := utils.GenerateOtp()

	err = repository.StroreOtpInRedis(ctx, rdb, email, otp)
	if err != nil {
		return errors.New("internal server error")
	}

	emailBody := fmt.Sprintf(utils.OtpBody, otp)

	err = utils.SendEmail(cfg, email, "your otp for verifying to tripneo", emailBody)
	if err != nil {
		log.Print(err)
		return errors.New("internal server error")
	}

	return nil
}

func ValidateOtp(ctx context.Context, rdb *redis.Client, email, otp string) error {
	user, err := repository.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, domainerrors.EmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return domainerrors.EmailNotFound
		}
		return errors.New("internal server error")
	}
	if user.IsVerified == true {
		return domainerrors.EmailALreadyVerified
	}
	err = repository.ValidateOtpInRedis(ctx, rdb, email, otp)
	if err != nil {
		if errors.Is(err, domainerrors.ErrInvalidOrExpiredOtp) {
			return domainerrors.ErrInvalidOrExpiredOtp
		}
		return errors.New("internal server error")
	}

	err = repository.UpdateUserVerified(email)
	if err != nil {
		return err
	}

	return nil
}

func ResendOtp(ctx context.Context, cfg *config.Config, rdb *redis.Client, email string) error {
	user, err := repository.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, domainerrors.EmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return domainerrors.EmailNotFound
		}
		return errors.New("internal server error")
	}
	if user.IsVerified == true {
		return domainerrors.EmailALreadyVerified
	}

	otp := utils.GenerateOtp()

	err = repository.ValidateAndStoreNewOtp(ctx, rdb, email, otp)
	if err != nil {
		if errors.Is(err, domainerrors.ErrOtpCooldownLimit) {
			return domainerrors.ResendOtpCooldown
		}
		return errors.New("internal server error")
	}

	emailBody := fmt.Sprintf(utils.OtpBody, otp)

	err = utils.SendEmail(cfg, email, "otp for verifying to tripneo", emailBody)
	if err != nil {
		log.Print(err)
		return errors.New("internal server error")
	}

	return nil
}

func Login(cfg *config.Config, email, password string) (string, *models.User, error) {
	user, err := repository.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, domainerrors.EmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return "", nil, domainerrors.EmailNotFound
		}
		return "", nil, errors.New("internal server error")
	}

	if user.IsVerified != true {
		return "", nil, domainerrors.VerifyEmailBeforeLoggingIN
	}

	err = utils.ValidatePassword(user.PasswordHash, password)
	if err != nil {
		return "", nil, domainerrors.InvalidEmailOrPassword
	}

	token, err := utils.GenerateToken(cfg, user.ID.String(), user.Role, user.Permissions)
	if err != nil {
		return "", nil, errors.New("internal server error")
	}

	return token, user, nil
}

func AssignRole(email string, role string, permissions []string) error {
	_, err := repository.FindUserByEmail(email)
	if err != nil {
		return err
	}
	return repository.AssignRole(email, role, permissions)
}
