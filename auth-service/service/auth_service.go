package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/junaid9001/tripneo/auth-service/config"
	"github.com/junaid9001/tripneo/auth-service/repository"
	"github.com/junaid9001/tripneo/auth-service/utils"
	"github.com/redis/go-redis/v9"
)

var EmailAlreadyTaken = errors.New("email already taken")
var InvalidOrExpiredOtp = errors.New("Invalid Otp")
var EmailALreadyVerified = errors.New("email already verified")
var ResendOtpCooldown = errors.New("please wait for 1 minute before requesting a new OTP")
var EmailNotFound = errors.New("email not found")
var InvalidEmailOrPassword = errors.New("invalid email or password")

func CreateUser(ctx context.Context, cfg *config.Config, rdb *redis.Client, email, password string) error {
	hashedPass, err := utils.GenerateHashedPassword(password)
	if err != nil {
		return errors.New("internal server error")
	}
	err = repository.InsertUser(email, hashedPass)
	if err != nil {
		if errors.Is(err, repository.ErrEmailALreadyTaken) {
			return EmailAlreadyTaken
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
		if errors.Is(err, repository.ErrEmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return EmailNotFound
		}
		return fmt.Errorf("Internal Server Error")
	}
	if user.IsVerified == true {
		return EmailALreadyVerified
	}
	err = repository.ValidateOtpInRedis(ctx, rdb, email, otp)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidOrExpiredOtp) {
			return InvalidOrExpiredOtp
		}
		return fmt.Errorf("Internal Server Error")
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
		if errors.Is(err, repository.ErrEmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return EmailNotFound
		}
		return fmt.Errorf("Internal Server Error")
	}
	if user.IsVerified == true {
		return EmailALreadyVerified
	}

	otp := utils.GenerateOtp()

	err = repository.ValidateAndStoreNewOtp(ctx, rdb, email, otp)
	if err != nil {
		if errors.Is(err, repository.ErrOtpCooldownLimit) {
			return ResendOtpCooldown
		}
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

func Login(cfg *config.Config, email, password string) (string, error) {
	user, err := repository.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, repository.ErrEmailNotFound) {
			//devlog
			log.Print("email mismatch or not found")
			return "", EmailNotFound
		}
		return "", fmt.Errorf("Internal Server Error")
	}

	err = utils.ValidatePassword(user.PasswordHash, password)
	if err != nil {
		return "", InvalidEmailOrPassword
	}

	token, err := utils.GenerateToken(cfg, user.ID.String(), user.Role)
	if err != nil {
		return "", fmt.Errorf("Internal Server Error")
	}

	return token, nil
}
