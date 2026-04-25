package redis

import (
	"context"
	"log"

	"github.com/Salman-kp/tripneo/bus-service/config"
	goredis "github.com/redis/go-redis/v9"
)

var Client *goredis.Client

func ConnectRedis(cfg *config.Config) {
	addr := cfg.REDIS_URL
	if addr == "" {
		addr = "localhost:6379"
	}

	Client = goredis.NewClient(&goredis.Options{
		Addr: addr,
	})

	// Test the connection
	if err := Client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Successfully connected to Redis")
}
