package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrInvalidOrExpiredOtp = errors.New("Invalid or Expired Otp")

var ErrOtpCooldownLimit = errors.New("please wait for 1 minute before requesting a new OTP")

func StroreOtpInRedis(ctx context.Context, rdb *redis.Client, email, otp string) error {
	key := "otp:" + email
	keyOtpCooldown := "otp:cooldown" + email
	err := rdb.Set(ctx, key, otp, 5*time.Minute).Err()
	if err != nil {
		log.Print(err)
		return err
	}

	err = rdb.Set(ctx, keyOtpCooldown, 1, 1*time.Minute).Err()
	if err != nil {
		log.Print(err)
		return err
	}

	return nil

}

// check cooldown and resend otp
func ValidateAndStoreNewOtp(ctx context.Context, rdb *redis.Client, email, otp string) error {
	key := "otp:" + email
	keyOtpCooldown := "otp:cooldown" + email

	err := rdb.Get(ctx, keyOtpCooldown).Err()
	if err != redis.Nil {

		return ErrOtpCooldownLimit
	}

	err = rdb.Set(ctx, key, otp, 5*time.Minute).Err()
	if err != nil {
		log.Print(err)
		return err
	}

	err = rdb.Set(ctx, keyOtpCooldown, 1, 1*time.Minute).Err()
	if err != nil {
		log.Print(err)
		return err

	}
	return nil
}

func ValidateOtpInRedis(ctx context.Context, rdb *redis.Client, email, enteredOtp string) error {
	key := "otp:" + email
	val, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrInvalidOrExpiredOtp
	}

	if err != nil {
		log.Print(err)
		return err
	}
	if val != enteredOtp {
		return ErrInvalidOrExpiredOtp
	}

	return nil
}
