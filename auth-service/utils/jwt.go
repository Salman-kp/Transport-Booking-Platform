package utils

import (
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/junaid9001/tripneo/auth-service/config"
)

type Claims struct {
	UserID string
	Role   string
	jwt.RegisteredClaims
}

func GenerateToken(cfg *config.Config, userID, role string) (string, error) {

	expiry, _ := time.ParseDuration(cfg.JWT_EXPIRY)
	log.Print(expiry)
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "tripneo",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(cfg.JWT_SECRET))
}
