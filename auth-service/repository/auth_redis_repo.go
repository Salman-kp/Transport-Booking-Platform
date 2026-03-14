package repository

import (
	"context"
	"log"
	"time"

	domainerrors "github.com/junaid9001/tripneo/auth-service/domain_errors"
	"github.com/redis/go-redis/v9"
)

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

	if err == nil {
		return domainerrors.ErrOtpCooldownLimit
	}
	if err != redis.Nil {
		log.Print(err)
		return err
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
		return domainerrors.ErrInvalidOrExpiredOtp
	}

	if err != nil {
		log.Print(err)
		return err
	}
	if val != enteredOtp {
		return domainerrors.ErrInvalidOrExpiredOtp
	}

	return nil
}
