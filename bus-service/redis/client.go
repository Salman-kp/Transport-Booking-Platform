package redis

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

// Client creates, verifies, and returns a connected Redis client.
func Client(redisHost, redisPort string) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
	})

	err := rdb.Set(context.Background(), "bus:health", "ok", 0).Err()
	if err != nil {
		log.Printf("❌ redis connection failed (bus-service): %v", err)
	}

	_, err = rdb.Get(context.Background(), "bus:health").Result()
	if err != nil {
		log.Printf("❌ redis get check failed (bus-service): %v", err)
	} else {
		log.Println("✅ redis connection succeeded (bus-service)")
	}

	return rdb
}
