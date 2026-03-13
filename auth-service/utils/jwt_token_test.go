package utils

import (
	"testing"

	"github.com/junaid9001/tripneo/auth-service/config"
)

func TestGenerateJwtToken(t *testing.T) {
	cfg := &config.Config{
		JWT_SECRET: "test-secret",
		JWT_EXPIRY: "60m",
	}
	_, err := GenerateToken(cfg, "1", "user")
	if err != nil {
		t.Fatalf("jwt token creation shouldnt return error, reson: %v", err)
	}

}
